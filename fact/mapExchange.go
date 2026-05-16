// Portions of the binary map exchange parser are adapted from
// rfvgyhn/factorio-exchange-string-parser.
//
// # MIT License
//
// # Copyright (c) 2024 rfvgyhn
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
package fact

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/util"
)

// MapExchangeData contains the JSON payloads Factorio accepts as map settings.
type MapExchangeData struct {
	Version        [4]uint16
	MapGenSettings map[string]interface{}
	MapSettings    map[string]interface{}
	Checksum       uint32
	ChecksumOK     bool
}

// ParseMapExchangeString converts a Factorio map exchange string into the two
// JSON settings tables expected by Factorio's --map-gen-settings and
// --map-settings command line flags.
//
// The binary field order follows Factorio's documented exchange string format
// and the MIT-licensed converter at:
// https://github.com/rfvgyhn/factorio-exchange-string-parser
func ParseMapExchangeString(input string) (*MapExchangeData, error) {
	trimmed := strings.TrimSpace(input)
	if strings.HasPrefix(trimmed, "{") {
		return parseMapExchangeJSON(trimmed)
	}

	compact := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, trimmed)

	if !strings.HasPrefix(compact, ">>>") || !strings.HasSuffix(compact, "<<<") {
		return nil, fmt.Errorf("invalid map exchange string: expected >>>...<<<")
	}

	encoded := strings.TrimSuffix(strings.TrimPrefix(compact, ">>>"), "<<<")
	decoded, err := decodeMapExchangeBase64(encoded)
	if err != nil {
		return nil, fmt.Errorf("invalid map exchange base64: %w", err)
	}

	zr, err := zlib.NewReader(bytes.NewReader(decoded))
	if err != nil {
		return nil, fmt.Errorf("invalid or unsupported map exchange compression: %w", err)
	}
	raw, err := io.ReadAll(zr)
	closeErr := zr.Close()
	if err != nil {
		return nil, fmt.Errorf("unable to decompress map exchange string: %w", err)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("unable to close map exchange decoder: %w", closeErr)
	}
	if len(raw) < 4 {
		return nil, fmt.Errorf("map exchange data is too short")
	}

	p := newMapExchangeParser(raw)
	version := p.readVersion()
	atLeastV2 := versionAtLeast(version, 2, 0, 0, 0)

	data := &MapExchangeData{
		Version:        version,
		MapGenSettings: nil,
		MapSettings:    nil,
	}
	_ = p.readUint8() // Unknown exchange-format byte.
	data.MapGenSettings = p.readMapGenSettings(atLeastV2)
	data.MapSettings = p.readMapSettings(atLeastV2)
	data.Checksum = p.readUint32()

	if p.err != nil {
		return nil, p.err
	}
	if p.pos != len(raw) {
		return nil, fmt.Errorf("unexpected data after map exchange payload: %d bytes", len(raw)-p.pos)
	}

	crcIndex := len(raw) - 4
	actual := binary.LittleEndian.Uint32(raw[crcIndex:])
	expected := crc32.ChecksumIEEE(raw[:crcIndex])
	data.ChecksumOK = actual == expected
	if !data.ChecksumOK {
		cwlog.DoLogCW("ParseMapExchangeString: checksum failed")
	}

	return data, nil
}

func parseMapExchangeJSON(input string) (*MapExchangeData, error) {
	var parsed struct {
		MapSettings       map[string]interface{} `json:"map_settings"`
		MapGenSettings    map[string]interface{} `json:"map_gen_settings"`
		MapSettingsCamel  map[string]interface{} `json:"mapSettings"`
		MapGenSettingsCam map[string]interface{} `json:"mapGenSettings"`
	}

	dec := json.NewDecoder(strings.NewReader(input))
	dec.UseNumber()
	if err := dec.Decode(&parsed); err != nil {
		return nil, fmt.Errorf("invalid map exchange JSON: %w", err)
	}

	if parsed.MapSettings == nil {
		parsed.MapSettings = parsed.MapSettingsCamel
	}
	if parsed.MapGenSettings == nil {
		parsed.MapGenSettings = parsed.MapGenSettingsCam
	}
	if parsed.MapSettings == nil || parsed.MapGenSettings == nil {
		return nil, fmt.Errorf("map exchange JSON must contain map_settings and map_gen_settings")
	}

	return &MapExchangeData{
		MapSettings:    parsed.MapSettings,
		MapGenSettings: parsed.MapGenSettings,
		ChecksumOK:     true,
	}, nil
}

func decodeMapExchangeBase64(encoded string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err == nil {
		return decoded, nil
	}

	padded := encoded
	switch len(padded) % 4 {
	case 2:
		padded += "=="
	case 3:
		padded += "="
	}
	if padded != encoded {
		if decoded, padErr := base64.StdEncoding.DecodeString(padded); padErr == nil {
			return decoded, nil
		}
	}

	decoded, rawErr := base64.RawStdEncoding.DecodeString(encoded)
	if rawErr == nil {
		return decoded, nil
	}
	return nil, err
}

// WriteCustomMapExchangeFiles writes the parsed exchange settings as the
// "custom" map generator pair used by GenNewMap.
func WriteCustomMapExchangeFiles(exchangeString string) (string, string, error) {
	data, err := ParseMapExchangeString(exchangeString)
	if err != nil {
		return "", "", err
	}

	genPath, setPath := cfg.GetMapGeneratorFiles(constants.CustomMapGeneratorName)
	dir := filepath.Dir(genPath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return "", "", fmt.Errorf("unable to create map generator directory: %w", err)
	}

	if err := util.WriteJSONAtomic(genPath, data.MapGenSettings, 0644); err != nil {
		return "", "", fmt.Errorf("unable to write custom map-gen settings: %w", err)
	}
	if err := util.WriteJSONAtomic(setPath, data.MapSettings, 0644); err != nil {
		return "", "", fmt.Errorf("unable to write custom map settings: %w", err)
	}

	return genPath, setPath, nil
}

// GenCustomMapFromExchange writes custom map settings, selects the custom
// generator in local config, then generates a new save using GenNewMap.
func GenCustomMapFromExchange(exchangeString string) (string, error) {
	if _, _, err := WriteCustomMapExchangeFiles(exchangeString); err != nil {
		return "", err
	}

	cfg.Local.Settings.MapGenerator = constants.CustomMapGeneratorName
	if !cfg.WriteLCfg() {
		return "", fmt.Errorf("unable to save cw-local config")
	}

	fileName, err := GenNewMap()
	if err != nil {
		return "", err
	}
	return fileName, nil
}

type mapExchangeParser struct {
	data         []byte
	pos          int
	err          error
	lastPosition mapExchangePosition
}

type mapExchangePosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

func newMapExchangeParser(data []byte) *mapExchangeParser {
	return &mapExchangeParser{data: data}
}

func (p *mapExchangeParser) setErr(format string, args ...interface{}) {
	if p.err == nil {
		p.err = fmt.Errorf(format, args...)
	}
}

func (p *mapExchangeParser) readBytes(n int) []byte {
	if p.err != nil {
		return nil
	}
	if n < 0 || p.pos+n > len(p.data) {
		p.setErr("map exchange data ended unexpectedly at byte %d", p.pos)
		return nil
	}
	out := p.data[p.pos : p.pos+n]
	p.pos += n
	return out
}

func (p *mapExchangeParser) readBool() bool {
	return p.readUint8() != 0
}

func (p *mapExchangeParser) readUint8() uint8 {
	b := p.readBytes(1)
	if b == nil {
		return 0
	}
	return b[0]
}

func (p *mapExchangeParser) readInt16() int16 {
	b := p.readBytes(2)
	if b == nil {
		return 0
	}
	return int16(binary.LittleEndian.Uint16(b))
}

func (p *mapExchangeParser) readUint16() uint16 {
	b := p.readBytes(2)
	if b == nil {
		return 0
	}
	return binary.LittleEndian.Uint16(b)
}

func (p *mapExchangeParser) readInt32() int32 {
	b := p.readBytes(4)
	if b == nil {
		return 0
	}
	return int32(binary.LittleEndian.Uint32(b))
}

func (p *mapExchangeParser) readUint32() uint32 {
	b := p.readBytes(4)
	if b == nil {
		return 0
	}
	return binary.LittleEndian.Uint32(b)
}

func (p *mapExchangeParser) readUint32SO() uint32 {
	value := p.readUint8()
	if value == 0xff {
		return p.readUint32()
	}
	return uint32(value)
}

func (p *mapExchangeParser) readFloat() float64 {
	return float64(math.Float32frombits(p.readUint32()))
}

func (p *mapExchangeParser) readDouble() float64 {
	b := p.readBytes(8)
	if b == nil {
		return 0
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(b))
}

func (p *mapExchangeParser) readString() string {
	size := p.readUint32SO()
	if size > uint32(len(p.data)-p.pos) {
		p.setErr("map exchange string length %d exceeds remaining data", size)
		return ""
	}
	b := p.readBytes(int(size))
	if b == nil {
		return ""
	}
	return string(b)
}

func (p *mapExchangeParser) readVersion() [4]uint16 {
	return [4]uint16{p.readUint16(), p.readUint16(), p.readUint16(), p.readUint16()}
}

func versionAtLeast(version [4]uint16, major, minor, patch, dev uint16) bool {
	target := [4]uint16{major, minor, patch, dev}
	for i := 0; i < len(version); i++ {
		if version[i] > target[i] {
			return true
		}
		if version[i] < target[i] {
			return false
		}
	}
	return true
}

func (p *mapExchangeParser) readOptional(readValue func() interface{}) interface{} {
	if !p.readBool() {
		return nil
	}
	return readValue()
}

func (p *mapExchangeParser) readArray(readItem func() interface{}) []interface{} {
	size := p.readUint32SO()
	if size > 100000 {
		p.setErr("map exchange array length %d is too large", size)
		return nil
	}
	out := make([]interface{}, 0, int(size))
	for i := uint32(0); i < size && p.err == nil; i++ {
		out = append(out, readItem())
	}
	return out
}

func (p *mapExchangeParser) readStringArray() []interface{} {
	return p.readArray(func() interface{} {
		return p.readString()
	})
}

func (p *mapExchangeParser) readStringDict(readValue func() interface{}) map[string]interface{} {
	size := p.readUint32SO()
	if size > 100000 {
		p.setErr("map exchange dictionary length %d is too large", size)
		return nil
	}
	out := make(map[string]interface{}, int(size))
	for i := uint32(0); i < size && p.err == nil; i++ {
		key := p.readString()
		out[key] = readValue()
	}
	return out
}

func (p *mapExchangeParser) readFrequencySizeRichness() interface{} {
	return map[string]interface{}{
		"frequency": p.readFloat(),
		"size":      p.readFloat(),
		"richness":  p.readFloat(),
	}
}

func (p *mapExchangeParser) readAutoplaceSetting() interface{} {
	return map[string]interface{}{
		"treat_missing_as_default": p.readBool(),
		"settings": p.readStringDict(func() interface{} {
			return p.readFrequencySizeRichness()
		}),
	}
}

func (p *mapExchangeParser) readMapPosition() interface{} {
	var x, y float64
	xDiff := float64(p.readInt16()) / 256
	if xDiff == float64(0x7fff)/256 {
		x = float64(p.readInt32()) / 256
		y = float64(p.readInt32()) / 256
	} else {
		yDiff := float64(p.readInt16()) / 256
		x = p.lastPosition.X + xDiff
		y = p.lastPosition.Y + yDiff
	}

	p.lastPosition.X = x
	p.lastPosition.Y = y

	return map[string]interface{}{
		"x": x,
		"y": y,
	}
}

func (p *mapExchangeParser) readBoundingBox() interface{} {
	return map[string]interface{}{
		"left_top":     p.readMapPosition(),
		"right_bottom": p.readMapPosition(),
		"orientation": map[string]interface{}{
			"x": p.readInt16(),
			"y": p.readInt16(),
		},
	}
}

func (p *mapExchangeParser) readCliffSettings(atLeastV2 bool) interface{} {
	settings := map[string]interface{}{
		"name": p.readString(),
	}
	if atLeastV2 {
		_ = p.readUint8() // New 2.x field not represented in JSON settings.
	}
	settings["cliff_elevation_0"] = p.readFloat()
	settings["cliff_elevation_interval"] = p.readFloat()
	settings["richness"] = p.readFloat()
	if atLeastV2 {
		settings["cliff_smoothing"] = p.readFloat()
	}
	return settings
}

func (p *mapExchangeParser) readTerritorySettings() interface{} {
	return map[string]interface{}{
		"units":                          p.readStringArray(),
		"territory_index_expression":     p.readString(),
		"territory_variation_expression": p.readString(),
		"minimum_territory_size":         p.readUint32(),
	}
}

func (p *mapExchangeParser) readMapGenSettings(atLeastV2 bool) map[string]interface{} {
	terrainSegmentation := float64(0)
	water := float64(0)
	if !atLeastV2 {
		terrainSegmentation = p.readFloat()
		water = p.readFloat()
	}

	settings := map[string]interface{}{
		"autoplace_controls": p.readStringDict(func() interface{} {
			return p.readFrequencySizeRichness()
		}),
		"autoplace_settings": p.readStringDict(func() interface{} {
			return p.readAutoplaceSetting()
		}),
		"default_enable_all_autoplace_controls": p.readBool(),
		"seed":                                  p.readUint32(),
		"width":                                 p.readUint32(),
		"height":                                p.readUint32(),
		"area_to_generate_at_start":             p.readBoundingBox(),
		"starting_area":                         p.readFloat(),
		"peaceful_mode":                         p.readBool(),
		"starting_points":                       nil,
		"property_expression_names":             nil,
		"cliff_settings":                        nil,
	}
	if atLeastV2 {
		settings["no_enemies_mode"] = p.readBool()
	}
	settings["starting_points"] = p.readArray(func() interface{} {
		return p.readMapPosition()
	})
	settings["property_expression_names"] = p.readStringDict(func() interface{} {
		return p.readString()
	})
	settings["cliff_settings"] = p.readCliffSettings(atLeastV2)
	if atLeastV2 {
		territorySettings := p.readOptional(func() interface{} {
			return p.readTerritorySettings()
		})
		if territorySettings != nil {
			settings["territory_settings"] = territorySettings
		}
	} else {
		settings["terrain_segmentation"] = terrainSegmentation
		settings["water"] = water
	}
	return settings
}

func (p *mapExchangeParser) readPollution() interface{} {
	return map[string]interface{}{
		"enabled":                                     p.readOptional(func() interface{} { return p.readBool() }),
		"diffusion_ratio":                             p.readOptional(func() interface{} { return p.readDouble() }),
		"min_to_diffuse":                              p.readOptional(func() interface{} { return p.readDouble() }),
		"ageing":                                      p.readOptional(func() interface{} { return p.readDouble() }),
		"expected_max_per_chunk":                      p.readOptional(func() interface{} { return p.readDouble() }),
		"min_to_show_per_chunk":                       p.readOptional(func() interface{} { return p.readDouble() }),
		"min_pollution_to_damage_trees":               p.readOptional(func() interface{} { return p.readDouble() }),
		"pollution_with_max_forest_damage":            p.readOptional(func() interface{} { return p.readDouble() }),
		"pollution_per_tree_damage":                   p.readOptional(func() interface{} { return p.readDouble() }),
		"pollution_restored_per_tree_damage":          p.readOptional(func() interface{} { return p.readDouble() }),
		"max_pollution_to_restore_trees":              p.readOptional(func() interface{} { return p.readDouble() }),
		"enemy_attack_pollution_consumption_modifier": p.readOptional(func() interface{} { return p.readDouble() }),
	}
}

func (p *mapExchangeParser) readRealSteering() interface{} {
	return map[string]interface{}{
		"radius":                         p.readOptional(func() interface{} { return p.readDouble() }),
		"separation_factor":              p.readOptional(func() interface{} { return p.readDouble() }),
		"separation_force":               p.readOptional(func() interface{} { return p.readDouble() }),
		"force_unit_fuzzy_goto_behavior": p.readOptional(func() interface{} { return p.readBool() }),
	}
}

func (p *mapExchangeParser) readSteering() interface{} {
	return map[string]interface{}{
		"default": p.readRealSteering(),
		"moving":  p.readRealSteering(),
	}
}

func (p *mapExchangeParser) readEnemyEvolution() interface{} {
	return map[string]interface{}{
		"enabled":          p.readOptional(func() interface{} { return p.readBool() }),
		"time_factor":      p.readOptional(func() interface{} { return p.readDouble() }),
		"destroy_factor":   p.readOptional(func() interface{} { return p.readDouble() }),
		"pollution_factor": p.readOptional(func() interface{} { return p.readDouble() }),
	}
}

func (p *mapExchangeParser) readEnemyExpansion() interface{} {
	return map[string]interface{}{
		"enabled":                             p.readOptional(func() interface{} { return p.readBool() }),
		"max_expansion_distance":              p.readOptional(func() interface{} { return p.readUint32() }),
		"friendly_base_influence_radius":      p.readOptional(func() interface{} { return p.readUint32() }),
		"enemy_building_influence_radius":     p.readOptional(func() interface{} { return p.readUint32() }),
		"building_coefficient":                p.readOptional(func() interface{} { return p.readDouble() }),
		"other_base_coefficient":              p.readOptional(func() interface{} { return p.readDouble() }),
		"neighbouring_chunk_coefficient":      p.readOptional(func() interface{} { return p.readDouble() }),
		"neighbouring_base_chunk_coefficient": p.readOptional(func() interface{} { return p.readDouble() }),
		"max_colliding_tiles_coefficient":     p.readOptional(func() interface{} { return p.readDouble() }),
		"settler_group_min_size":              p.readOptional(func() interface{} { return p.readUint32() }),
		"settler_group_max_size":              p.readOptional(func() interface{} { return p.readUint32() }),
		"min_expansion_cooldown":              p.readOptional(func() interface{} { return p.readUint32() }),
		"max_expansion_cooldown":              p.readOptional(func() interface{} { return p.readUint32() }),
	}
}

func (p *mapExchangeParser) readUnitGroup() interface{} {
	return map[string]interface{}{
		"min_group_gathering_time":           p.readOptional(func() interface{} { return p.readUint32() }),
		"max_group_gathering_time":           p.readOptional(func() interface{} { return p.readUint32() }),
		"max_wait_time_for_late_members":     p.readOptional(func() interface{} { return p.readUint32() }),
		"max_group_radius":                   p.readOptional(func() interface{} { return p.readDouble() }),
		"min_group_radius":                   p.readOptional(func() interface{} { return p.readDouble() }),
		"max_member_speedup_when_behind":     p.readOptional(func() interface{} { return p.readDouble() }),
		"max_member_slowdown_when_ahead":     p.readOptional(func() interface{} { return p.readDouble() }),
		"max_group_slowdown_factor":          p.readOptional(func() interface{} { return p.readDouble() }),
		"max_group_member_fallback_factor":   p.readOptional(func() interface{} { return p.readDouble() }),
		"member_disown_distance":             p.readOptional(func() interface{} { return p.readDouble() }),
		"tick_tolerance_when_member_arrives": p.readOptional(func() interface{} { return p.readUint32() }),
		"max_gathering_unit_groups":          p.readOptional(func() interface{} { return p.readUint32() }),
		"max_unit_group_size":                p.readOptional(func() interface{} { return p.readUint32() }),
	}
}

func (p *mapExchangeParser) readPathFinder() interface{} {
	return map[string]interface{}{
		"fwd2bwd_ratio":                                        p.readOptional(func() interface{} { return p.readInt32() }),
		"goal_pressure_ratio":                                  p.readOptional(func() interface{} { return p.readDouble() }),
		"use_path_cache":                                       p.readOptional(func() interface{} { return p.readBool() }),
		"max_steps_worked_per_tick":                            p.readOptional(func() interface{} { return p.readDouble() }),
		"max_work_done_per_tick":                               p.readOptional(func() interface{} { return p.readUint32() }),
		"short_cache_size":                                     p.readOptional(func() interface{} { return p.readUint32() }),
		"long_cache_size":                                      p.readOptional(func() interface{} { return p.readUint32() }),
		"short_cache_min_cacheable_distance":                   p.readOptional(func() interface{} { return p.readDouble() }),
		"short_cache_min_algo_steps_to_cache":                  p.readOptional(func() interface{} { return p.readUint32() }),
		"long_cache_min_cacheable_distance":                    p.readOptional(func() interface{} { return p.readDouble() }),
		"cache_max_connect_to_cache_steps_multiplier":          p.readOptional(func() interface{} { return p.readUint32() }),
		"cache_accept_path_start_distance_ratio":               p.readOptional(func() interface{} { return p.readDouble() }),
		"cache_accept_path_end_distance_ratio":                 p.readOptional(func() interface{} { return p.readDouble() }),
		"negative_cache_accept_path_start_distance_ratio":      p.readOptional(func() interface{} { return p.readDouble() }),
		"negative_cache_accept_path_end_distance_ratio":        p.readOptional(func() interface{} { return p.readDouble() }),
		"cache_path_start_distance_rating_multiplier":          p.readOptional(func() interface{} { return p.readDouble() }),
		"cache_path_end_distance_rating_multiplier":            p.readOptional(func() interface{} { return p.readDouble() }),
		"stale_enemy_with_same_destination_collision_penalty":  p.readOptional(func() interface{} { return p.readDouble() }),
		"ignore_moving_enemy_collision_distance":               p.readOptional(func() interface{} { return p.readDouble() }),
		"enemy_with_different_destination_collision_penalty":   p.readOptional(func() interface{} { return p.readDouble() }),
		"general_entity_collision_penalty":                     p.readOptional(func() interface{} { return p.readDouble() }),
		"general_entity_subsequent_collision_penalty":          p.readOptional(func() interface{} { return p.readDouble() }),
		"extended_collision_penalty":                           p.readOptional(func() interface{} { return p.readDouble() }),
		"max_clients_to_accept_any_new_request":                p.readOptional(func() interface{} { return p.readUint32() }),
		"max_clients_to_accept_short_new_request":              p.readOptional(func() interface{} { return p.readUint32() }),
		"direct_distance_to_consider_short_request":            p.readOptional(func() interface{} { return p.readUint32() }),
		"short_request_max_steps":                              p.readOptional(func() interface{} { return p.readUint32() }),
		"short_request_ratio":                                  p.readOptional(func() interface{} { return p.readDouble() }),
		"min_steps_to_check_path_find_termination":             p.readOptional(func() interface{} { return p.readUint32() }),
		"start_to_goal_cost_multiplier_to_terminate_path_find": p.readOptional(func() interface{} { return p.readDouble() }),
		"overload_levels": p.readOptional(func() interface{} {
			return p.readArray(func() interface{} { return p.readUint32() })
		}),
		"overload_multipliers": p.readOptional(func() interface{} {
			return p.readArray(func() interface{} { return p.readDouble() })
		}),
		"negative_path_cache_delay_interval": p.readOptional(func() interface{} { return p.readUint32() }),
	}
}

func (p *mapExchangeParser) readDifficultySettings(atLeastV2 bool) interface{} {
	if atLeastV2 {
		return map[string]interface{}{
			"technology_price_multiplier": p.readDouble(),
			"spoil_time_modifier":         p.readDouble(),
		}
	}

	recipeDifficulty := p.readUint8()
	technologyDifficulty := p.readUint8()
	technologyPriceMultiplier := p.readDouble()
	researchQueue := []string{"always", "after-victory", "never"}
	queueIndex := int(p.readUint8())
	queueSetting := ""
	if queueIndex >= 0 && queueIndex < len(researchQueue) {
		queueSetting = researchQueue[queueIndex]
	} else {
		p.setErr("invalid research queue setting %d", queueIndex)
	}

	return map[string]interface{}{
		"recipe_difficulty":           recipeDifficulty,
		"technology_difficulty":       technologyDifficulty,
		"technology_price_multiplier": technologyPriceMultiplier,
		"research_queue_setting":      queueSetting,
	}
}

func (p *mapExchangeParser) readAsteroidsSettings() interface{} {
	return map[string]interface{}{
		"spawning_rate":                     p.readOptional(func() interface{} { return p.readDouble() }),
		"max_ray_portals_expanded_per_tick": p.readOptional(func() interface{} { return p.readUint32() }),
	}
}

func (p *mapExchangeParser) readMapSettings(atLeastV2 bool) map[string]interface{} {
	settings := map[string]interface{}{
		"pollution":                 p.readPollution(),
		"steering":                  p.readSteering(),
		"enemy_evolution":           p.readEnemyEvolution(),
		"enemy_expansion":           p.readEnemyExpansion(),
		"unit_group":                p.readUnitGroup(),
		"path_finder":               p.readPathFinder(),
		"max_failed_behavior_count": p.readUint32(),
		"difficulty_settings":       p.readDifficultySettings(atLeastV2),
	}
	if atLeastV2 {
		settings["asteroids"] = p.readAsteroidsSettings()
	}
	return settings
}
