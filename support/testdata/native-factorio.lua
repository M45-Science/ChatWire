-- Appended to a temporary copy of SoftMod/control.lua by the Go integration test.
-- All players, GUIs, entities, serialization and events here use the real engine.
local function check(value, message)
    assert(value, "native Factorio: " .. message)
end

commands.add_command("native-save-marker", "Integration test only", function()
    storage.native_save_marker = "persisted"
end)

local function player_fixture()
    local player = game.get_player(1)
    check(player and player.connected, "test mode must provide a connected player")
    check(player.name == "NativeTester", "isolated player-data username not loaded")
    if not player.character then player.create_character() end
    STORAGE_MakePlayerStorage(player)
    player.admin = false
    PERMS_MakeNew(nil, player)
    return player
end

local cleanup_fixture
script.on_nth_tick(1, function()
    if not script.test_arg then return end -- ordinary headless server lifecycle
    local args = helpers.json_to_table(script.test_arg)
    if args.mode == "reload" then
        check(storage.native_save_marker == "persisted", "quit-save lost storage")
        check(not storage.CW_MapPreviewPending, "reload rearmed new-map announcement")
        CW_AnnounceNewMapPreview()
    else
        local player = player_fixture()
        if args.mode == "export" then
            storage.SM_Store.quickbar_presets = true
            QUICKBAR_MakeExchangeButton(player)
            QUICKBAR_Clicks{player_index = player.index, element = player.gui.top.qb_exchange_button}
            local pending = storage.PData[player.index].qb_pending
            check(pending and pending.action == "access", "manager did not request access")
            QUICKBAR_Result{name = player.name, id = pending.id, allowed = true, labels = {"", ""}}
            player.set_quick_bar_slot(1, 1, "transport-belt")
            player.set_quick_bar_slot(2, 1, {type = "filter", filter = {name = "inserter", quality = "uncommon"}})
            player.set_quick_bar_slot(7, 1, "electric-mining-drill")
            player.set_quick_bar_slot(10, 10, "rail")
            local row = player.gui.screen.quickbar_manager.qb_preset_section.qb_presets
            row.slot.selected_index = 2
            QUICKBAR_Clicks{player_index = player.index, element = row.store}
            check(storage.PData[player.index].qb_pending.action == "store", "GUI did not store")
        elseif args.mode == "import" then
            storage.SM_Store.quickbar_presets = true
            local pdata = storage.PData[player.index]
            QUICKBAR_MakeExchangeButton(player)
            QUICKBAR_Clicks{player_index = player.index, element = player.gui.top.qb_exchange_button}
            QUICKBAR_Result{name = player.name, id = pdata.qb_pending.id, allowed = true, labels = args.reply.labels}
            for page = 1, 10 do
                for slot = 1, 10 do player.set_quick_bar_slot(page, slot, nil) end
            end
            local row = player.gui.screen.quickbar_manager.qb_preset_section.qb_presets
            row.slot.selected_index = 2
            QUICKBAR_Clicks{player_index = player.index, element = row.load}
            check(pdata.qb_pending.action == "load", "GUI did not request load")
            args.reply.name = player.name
            args.reply.id = pdata.qb_pending.id
            QUICKBAR_Result(args.reply)
            check(not pdata.qb_pending, "load did not finish")
            check(ExportQuickbar(player, false) == args.reply.exchange, "persisted toolbar changed")
            check(player.get_quick_bar_slot(2, 1).filter.quality == "uncommon", "quality lost")
            check(player.get_quick_bar_slot(10, 10).filter.name == "rail", "last slot lost")
        elseif args.mode == "deconstruction" then
            if not cleanup_fixture then
                local surface = game.create_surface("native-cleanup", {width = 256, height = 256})
                surface.request_to_generate_chunks({0, 0}, 2)
                surface.force_generate_chunk_requests()
                for _, entity in pairs(surface.find_entities_filtered{type = "resource"}) do entity.destroy() end
                local tiles = {}
                for x = -40, 40 do
                    for y = -40, 40 do tiles[#tiles + 1] = {name = "grass-1", position = {x, y}} end
                end
                surface.set_tiles(tiles)
                for _, entity in pairs(surface.find_entities_filtered{force = "neutral"}) do
                    if entity.type ~= "character" then entity.destroy() end
                end
                local depleted = {}
                for i = 1, 64 do
                    local drill = surface.create_entity{name = "electric-mining-drill", position = {-30 + (i % 8) * 4, -30 + math.floor(i / 8) * 4}, force = "player"}
                    check(drill and not drill.mining_target, "empty drill fixture invalid")
                    drill.energy = 1000000
                    depleted[#depleted + 1] = drill
                end
                surface.create_entity{name = "iron-ore", position = {20, 20}, amount = 1000}
                local live = surface.create_entity{name = "electric-mining-drill", position = {20, 20}, force = "player"}
                check(live and live.energy == 0, "live drill must lack power")
                local chests = {}
                for i = 1, 64 do
                    local chest = surface.create_entity{name = "steel-chest", position = {10 + i % 8, -20 + math.floor(i / 8)}, force = "player"}
                    check(chest, "chest not created")
                    chests[#chests + 1] = chest
                end
                cleanup_fixture = {surface = surface, depleted = depleted, live = live, chests = chests}
                return -- let real drill/resource state update before using the planner
            end
            local surface, depleted = cleanup_fixture.surface, cleanup_fixture.depleted
            local live, chests = cleanup_fixture.live, cleanup_fixture.chests
            CW_EmitEvent("native-phase", {name = "depleted", player = player.name})
            if prototypes.entity["depleted-mining-drill-proxy"] then
                -- 2.1.25: use the actual new planner filter over a mixed area.
                local inventory = game.create_inventory(1)
                local planner = inventory[1]
                planner.set_stack{name = "deconstruction-planner"}
                planner.entity_filter_mode = defines.deconstruction_item.entity_filter_mode.whitelist
                check(planner.set_entity_filter(1, "depleted-mining-drill-proxy"), "depleted planner filter rejected")
                planner.deconstruct_area{surface = surface, force = player.force, area = {{-40, -40}, {40, 40}}, by_player = player}
                inventory.destroy()
                print("NATIVE-DEPLETED-PLANNER-PASS")
            else
                -- 2.1.24: the filter did not exist, but entity events are equivalent.
                for _, drill in ipairs(depleted) do drill.order_deconstruction(player.force, player) end
            end
            for _, drill in ipairs(depleted) do check(drill.to_be_deconstructed(), "depleted drill not marked (status " .. drill.status .. ")") end
            check(not live.to_be_deconstructed(), "depleted filter marked a live powerless drill")
            for _, chest in ipairs(chests) do check(not chest.to_be_deconstructed(), "depleted filter marked a chest") end
            CW_EmitEvent("native-phase", {name = "live", player = player.name})
            check(live and live.order_deconstruction(player.force, player), "live drill not marked")
            CW_EmitEvent("native-phase", {name = "structures", player = player.name})
            for _, chest in ipairs(chests) do
                check(chest and chest.order_deconstruction(player.force, player), "chest not marked")
            end
            CW_EmitEvent("native-phase", {name = "cancel-depleted", player = player.name})
            for _, drill in ipairs(depleted) do drill.cancel_deconstruction(player.force, player) end
        else
            error("unknown native test mode")
        end
    end
    print("NATIVE-FACTORIO-PASS " .. args.mode)
    script.finish_test(true)
end)
