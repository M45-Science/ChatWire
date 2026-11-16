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
        local player = args.mode == "essentials" and storage.native_hint_started and game.get_player(1) or player_fixture()
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
        elseif args.mode == "essentials" then
            local pdata = storage.PData[player.index]
            if not storage.native_hint_started then
                check(not player.gui.screen.m45_info_window, "Help/Info still opens on join")
                ESSENTIALS_Join(player)
                storage.SM_Store.tool_hints = true
                storage.native_hint_started = game.tick
            end
            local elapsed = game.tick - storage.native_hint_started
            local s = pdata.essentials
            if elapsed < 60 then
                check(s.waiting and not s.pending, "idle join started the reminder timer")
                check(not player.gui.screen.m45_essentials, "idle join opened panel")
                return
            end
            if elapsed == 60 then
                ESSENTIALS_Activity{player_index = player.index, name = defines.events.on_built_entity}
                check(not s.waiting and s.remaining == 480, "first action did not start eight-second timer")
            end
            if elapsed < 540 then
                check(not s.pending and not player.gui.screen.m45_essentials, "panel opened before eight seconds")
                return
            end
            if elapsed == 540 then
                local dialog = player.gui.screen.add{type = "frame", name = "native_hint_busy"}
                player.opened = dialog
            end
            if elapsed <= 590 then
                check(not s.pending, "hint interrupted an open interface")
                if elapsed == 590 then
                    player.opened = nil
                    player.gui.screen.native_hint_busy.destroy()
                end
                return
            end
            if not s.pending then return end
            local request = s.pending
            ESSENTIALS_Result{name = player.name, id = request + 1, allowed = true}
            check(s.pending == request and not s.ready, "stale response accepted")
            ESSENTIALS_Result{name = player.name, id = request, allowed = true}
            ESSENTIALS_Tick()
            local panel = player.gui.screen.m45_essentials
            check(panel and not player.opened, "panel missing or stole keyboard focus")
            check(panel.location.x > player.display_resolution.width / 2, "panel is not on right edge")
            check(panel.style.maximal_width <= 640, "panel exceeds width cap")
            check(panel.body.type == "flow", "essentials should not have a scrollbar")
            check(panel.body.grid.column_count == 2, "essentials should use two columns")
            local first = panel.body.grid.children[1]
            for _, card in pairs(panel.body.grid.children) do
                check(card.style.minimal_width == first.style.minimal_width and card.style.minimal_height == first.style.minimal_height, "grid cards have different sizes")
            end
            local home_height = first.style.minimal_height
            local children = panel.body.grid.children
            check(children[#children - 1].name == "essentials_discord_section", "Discord is not immediately before Patreon")
            check(children[#children].name == "essentials_patreon_section", "Patreon is not the last essentials button")
            check(panel.body.children[2].name == "credits", "supporter credits do not follow the grid")
            check(panel.titlebar.logo.style.stretch_image_to_widget_size, "title logo is not scaled")
            check(not panel.actions.essentials_help.visible, "home page has a redundant Back button")
            storage.SM_Store.patreonCredits = {"Zed", "Alice", "ALICE", "[color=red]Name"}
            ESSENTIALS_RefreshCredits(player)
            check(panel.body.credits.names.caption == "[color=red]Name, Alice, Zed", "supporter names were not sorted and deduplicated")
            check(panel.body.credits.names.style.rich_text_setting == defines.rich_text_setting.disabled, "supporter names allow markup")
            ESSENTIALS_Click{player_index = player.index, element = panel.body.grid.essentials_membership_section.essentials_membership}
            check(panel.body.membership_page.visible and not panel.body.grid.visible and not panel.body.credits.visible, "membership did not open on its own page")
            check(panel.body.membership_page.status.caption:find("Your level: New", 1, true), "membership page did not show current rank")
            check(#panel.body.membership_page.levels.children == 8, "membership page is missing a level")
            check(panel.body.membership_page.banish.essentials_membership_players.sprite == "file/img/buttons/online-64.png", "Banish instructions do not show the Players toolbar icon")
            ESSENTIALS_Click{player_index = player.index, element = panel.body.membership_page.essentials_membership_discord}
            check(panel.body.discord_page.visible and not panel.body.membership_page.visible, "Discord did not open on its own page")
            check(panel.body.discord_page.invite.read_only, "Discord invite is not read-only")
            ESSENTIALS_Click{player_index = player.index, element = panel.actions.essentials_help}
            check(panel.body.membership_page.visible, "Back from Discord did not restore membership")
            ESSENTIALS_Click{player_index = player.index, element = panel.actions.essentials_help}
            check(panel.body.grid.visible and not panel.body.membership_page.visible, "Back did not restore essentials")
            ESSENTIALS_Click{player_index = player.index, element = panel.body.grid.essentials_patreon_section.essentials_patreon}
            check(panel.body.patreon_page.visible and not panel.body.grid.visible, "Patreon did not open on its own page")
            check(panel.body.patreon_page.invite.read_only, "Patreon link is not read-only")
            ESSENTIALS_Click{player_index = player.index, element = panel.actions.essentials_help}
            check(panel.body.grid.visible, "Back from Patreon did not restore the grid")
            for _, card in pairs(panel.body.grid.children) do
                check(card.style.minimal_height == home_height and #card.children == 2, "opening a detail page changed the main grid")
            end
            ESSENTIALS_Click{player_index = player.index, element = panel.body.grid.essentials_todo_section.essentials_todo}
            check(not player.gui.screen.m45_essentials and player.gui.screen.m45_todo, "to-do action failed")
            player.gui.screen.m45_todo.destroy()
            ESSENTIALS_Open(player)
            panel = player.gui.screen.m45_essentials
            ESSENTIALS_Click{player_index = player.index, element = panel.body.grid.essentials_players_section.essentials_players}
            check(not player.gui.screen.m45_essentials and player.gui.left.m45_online, "Players action failed")
            player.gui.left.m45_online.destroy()
            ESSENTIALS_Open(player)
            panel = player.gui.screen.m45_essentials
            ESSENTIALS_Click{player_index = player.index, element = panel.body.grid.essentials_membership_section.essentials_membership}
            ESSENTIALS_Click{player_index = player.index, element = panel.body.membership_page.banish.essentials_membership_players}
            check(not player.gui.screen.m45_essentials and player.gui.left.m45_online, "Players icon in Banish instructions did not open the player list")
            player.gui.left.m45_online.destroy()
            -- A fresh join creates a new session; delayed replies from the old session are ignored.
            ESSENTIALS_Join(player)
            ESSENTIALS_Result{name = player.name, id = request, allowed = true}
            check(not pdata.essentials.ready, "old-session response accepted")
            local fresh = pdata.essentials
            fresh.pending = request + 1
            ESSENTIALS_Result{name = player.name, id = fresh.pending, allowed = false}
            ESSENTIALS_Tick()
            check(not player.gui.screen.m45_essentials, "cooldown denial opened panel")
            fresh.pending = request + 2
            ESSENTIALS_Result{name = player.name, id = fresh.pending, allowed = true}
            ESSENTIALS_Tick()
            panel = player.gui.screen.m45_essentials
            ESSENTIALS_Click{player_index = player.index, element = panel.body.grid.essentials_toolbar_section.essentials_toolbar}
            check(player.gui.screen.quickbar_manager and not player.gui.screen.m45_essentials, "toolbar action failed")
            QUICKBAR_OpenManager(player) -- Opening twice should reuse the same window.
            player.gui.screen.quickbar_manager.destroy()
            fresh.pending = request + 3
            ESSENTIALS_Result{name = player.name, id = fresh.pending, allowed = true}
            ESSENTIALS_Tick()
            panel = player.gui.screen.m45_essentials
            ESSENTIALS_Click{player_index = player.index, element = panel.actions.essentials_close}
            check(not player.gui.screen.m45_essentials, "Got it did not close panel")
            ESSENTIALS_Leave(player)
            check(not pdata.essentials, "disconnect retained pending introduction")
            INFO_Clicks{player_index = player.index, element = player.gui.top.m45_button}
            check(player.gui.screen.m45_essentials and not player.gui.screen.m45_info_window, "Help/Info did not open the compact panel")
            check(not pdata.essentials.waiting and not pdata.essentials.pending, "manual Help/Info rearmed automatic reminders")
            INFO_Clicks{player_index = player.index, element = player.gui.top.m45_button}
            check(not player.gui.screen.m45_essentials, "Help/Info button did not toggle closed")
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
