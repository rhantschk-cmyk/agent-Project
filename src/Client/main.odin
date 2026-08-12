package main

import "base:runtime"
import "core:encoding/json"
import "core:fmt"
import "core:os"
import "core:strings"
import "core:thread"
import "vendor:curl"
import "vendor:raylib"

// --- CONFIG STRUCT ---
Config :: struct {
	server_ip:  string,
	verify_key: string,
}

// --- MONITORING DATA STRUCT ---
SystemStats :: struct {
	cpu_usage_percent: f64,
	ram_usage_percent: f64,
	ram_total_mb:      u64,
	ram_used_mb:       u64,
	disk_free_gb:      u64,
	disk_total_gb:     u64,
	uptime_seconds:    u64,
	uptime_formated:   string,
	gpu_info:          string,
}

// --- TAB ENUM ---
Tab :: enum {
	Chat,
	Monitoring,
}

// --- ZUSTAND DER APP ---
App_State :: struct {
	config:          Config,
	current_tab:     Tab,

	// UI & Text-Eingabe (Chat)
	input_buf:       [256]u8,
	input_len:       int,
	agent_response:  string,
	is_loading_chat: bool,

	// Monitoring
	stats:           SystemStats,
	has_stats:       bool,
	is_loading_mon:  bool,
	last_mon_tick:   f32,
}

state: App_State

// --- CONFIG LADER ---
load_config :: proc() -> Config {
	default_cfg := Config {
		server_ip  = "localhost",
		verify_key = "abcdefgh",
	}

	data, err := os.read_entire_file_from_path("config.json", context.allocator)
	if err != nil {
		fmt.println("-> [CONFIG] 'config.json' nicht gefunden, verwende Standardwerte.")
		return default_cfg
	}
	defer delete(data)

	cfg: Config
	err_unmarshall := json.unmarshal(data, &cfg)
	if err_unmarshall != nil {
		fmt.printfln(
			"-> [CONFIG] Fehler beim Lesen der config.json: %v. Verwende Standardwerte.",
			err,
		)
		return default_cfg
	}

	fmt.printfln("-> [CONFIG] Geladen: IP=%s | Key=%s", cfg.server_ip, cfg.verify_key)
	return cfg
}

// --- HELFER: HTTP POST REQUEST ---
http_post :: proc(url: cstring, json_payload: cstring) -> string {
	curl.global_init(curl.GLOBAL_ALL)
	defer curl.global_cleanup()

	handle := curl.easy_init()
	if handle == nil do return ""
	defer curl.easy_cleanup(handle)

	Response_Buffer :: struct {
		data: strings.Builder,
	}
	buf: Response_Buffer
	strings.builder_init(&buf.data)
	defer strings.builder_destroy(&buf.data)

	write_cb :: proc "c" (ptr: rawptr, size: u64, nmemb: u64, userdata: rawptr) -> u64 {
		context = runtime.default_context()
		total := size * nmemb
		b := (^Response_Buffer)(userdata)
		slice := ([^]u8)(ptr)[:total]
		strings.write_bytes(&b.data, slice)
		return total
	}

	headers: ^curl.slist = nil
	headers = curl.slist_append(headers, "Content-Type: application/json")
	defer curl.slist_free_all(headers)

	curl.easy_setopt(handle, .URL, url)
	curl.easy_setopt(handle, .POST, 1)
	curl.easy_setopt(handle, .POSTFIELDS, json_payload)
	curl.easy_setopt(handle, .HTTPHEADER, headers)
	curl.easy_setopt(handle, .WRITEFUNCTION, write_cb)
	curl.easy_setopt(handle, .WRITEDATA, &buf)

	res := curl.easy_perform(handle)
	if res != .E_OK {
		return ""
	}

	result_str := strings.to_string(buf.data)
	return strings.clone(result_str)
}

// --- THREAD: AGENT CHAT (PORT 8080) ---
send_agent_thread :: proc() {
	prompt_str := string(state.input_buf[:state.input_len])

	Request :: struct {
		verify_key: string,
		prompt:     string,
	}

	Response :: struct {
		response: string,
		error:    string,
	}

	req := Request {
		verify_key = state.config.verify_key,
		prompt     = prompt_str,
	}

	json_data, err := json.marshal(req)
	if err != nil {
		state.agent_response = "Fehler beim Erstellen der JSON-Anfrage"
		state.is_loading_chat = false
		return
	}
	defer delete(json_data)

	c_payload := strings.clone_to_cstring(string(json_data))
	defer delete(c_payload)

	// URL dynamisch aus Server-IP aus Config bauen
	url_str := fmt.aprintf("http://%s:8080/api/agent/ask", state.config.server_ip)
	defer delete(url_str)
	c_url := strings.clone_to_cstring(url_str)
	defer delete(c_url)

	raw_resp := http_post(c_url, c_payload)
	defer delete(raw_resp)

	if len(raw_resp) == 0 {
		state.agent_response = fmt.aprintf("Verbindungsfehler zu %s:8080", state.config.server_ip)
		state.is_loading_chat = false
		return
	}

	resp_obj: Response
	unmarshal_err := json.unmarshal(transmute([]u8)raw_resp, &resp_obj)

	if unmarshal_err == nil && len(resp_obj.response) > 0 {
		state.agent_response = strings.clone(resp_obj.response)
	} else if len(resp_obj.error) > 0 {
		state.agent_response = fmt.aprintf("Fehler: %s", resp_obj.error)
	} else {
		state.agent_response = strings.clone(raw_resp)
	}

	state.is_loading_chat = false
}

// --- THREAD: MONITORING (PORT 9000) ---
fetch_monitoring_thread :: proc() {
	url_str := fmt.aprintf("http://%s:9000/api/stats", state.config.server_ip)
	defer delete(url_str)
	c_url := strings.clone_to_cstring(url_str)
	defer delete(c_url)

	raw_resp := http_post(c_url, "{}")
	defer delete(raw_resp)

	if len(raw_resp) > 0 {
		new_stats: SystemStats
		err := json.unmarshal(transmute([]u8)raw_resp, &new_stats)
		if err == nil {
			state.stats = new_stats
			state.has_stats = true
		}
	}
	state.is_loading_mon = false
}

// --- HELFER: MEHRZEILIGEN TEXT MIT WORD-WRAP ZEICHNEN ---
draw_wrapped_text :: proc(
	text: string,
	x, y, max_width: i32,
	font_size: i32,
	color: raylib.Color,
) {
	words := strings.split(text, " ")
	defer delete(words)

	curr_x := x
	curr_y := y
	line_height := font_size + 4

	for word in words {
		c_word := strings.clone_to_cstring(word)
		word_w := raylib.MeasureText(c_word, font_size)
		delete(c_word)

		if curr_x + word_w > x + max_width {
			curr_x = x
			curr_y += line_height
		}

		c_word_space := strings.clone_to_cstring(fmt.tprintf("%s ", word))
		raylib.DrawText(c_word_space, curr_x, curr_y, font_size, color)
		curr_x += raylib.MeasureText(c_word_space, font_size)
		delete(c_word_space)
	}
}

// --- MAIN / UI LOOP ---
main :: proc() {
	// Config laden
	state.config = load_config()

	raylib.InitWindow(850, 550, "Agent & System Dashboard")
	defer raylib.CloseWindow()
	raylib.SetTargetFPS(60)

	state.current_tab = .Chat
	state.agent_response = "Stelle eine Frage an den Agenten..."

	for !raylib.WindowShouldClose() {
		dt := raylib.GetFrameTime()
		state.last_mon_tick += dt

		// Auto-Refresh Monitoring alle 3 Sekunden
		if (state.last_mon_tick >= 3.0 || !state.has_stats) && !state.is_loading_mon {
			state.last_mon_tick = 0
			state.is_loading_mon = true
			thread.create_and_start(fetch_monitoring_thread)
		}

		// Tab-Wechsel per Tastatur
		if raylib.IsKeyPressed(.ONE) do state.current_tab = .Chat
		if raylib.IsKeyPressed(.TWO) do state.current_tab = .Monitoring
		if raylib.IsKeyPressed(.TAB) {
			if state.current_tab == .Chat do state.current_tab = .Monitoring
			else do state.current_tab = .Chat
		}

		// Chat Text-Eingabe (nur wenn im Chat Tab)
		if state.current_tab == .Chat {
			key := raylib.GetCharPressed()
			for key > 0 {
				if (key >= 32) && (key <= 125) && (state.input_len < len(state.input_buf) - 1) {
					state.input_buf[state.input_len] = u8(key)
					state.input_len += 1
				}
				key = raylib.GetCharPressed()
			}

			if raylib.IsKeyPressed(.BACKSPACE) && state.input_len > 0 {
				state.input_len -= 1
			}

			if raylib.IsKeyPressed(.ENTER) && state.input_len > 0 && !state.is_loading_chat {
				state.is_loading_chat = true
				state.agent_response = "Agent denkt nach..."
				thread.create_and_start(send_agent_thread)
			}
		}

		// --- DRAWING ---
		raylib.BeginDrawing()
		raylib.ClearBackground(raylib.Color{18, 18, 24, 255})

		// --- TAB HEADER BAR ---
		raylib.DrawRectangle(0, 0, 850, 45, raylib.Color{28, 28, 38, 255})

		// TAB 1: CHAT BUTTON
		btn_chat_col :=
			state.current_tab == .Chat ? raylib.Color{50, 50, 70, 255} : raylib.Color{35, 35, 48, 255}
		raylib.DrawRectangle(10, 8, 140, 30, btn_chat_col)
		raylib.DrawText("1: Agent Chat", 25, 14, 16, raylib.WHITE)
		if raylib.CheckCollisionPointRec(
			   raylib.GetMousePosition(),
			   raylib.Rectangle{10, 8, 140, 30},
		   ) &&
		   raylib.IsMouseButtonPressed(.LEFT) {
			state.current_tab = .Chat
		}

		// TAB 2: MONITORING BUTTON
		btn_mon_col :=
			state.current_tab == .Monitoring ? raylib.Color{50, 50, 70, 255} : raylib.Color{35, 35, 48, 255}
		raylib.DrawRectangle(160, 8, 150, 30, btn_mon_col)
		raylib.DrawText("2: Monitoring", 175, 14, 16, raylib.WHITE)
		if raylib.CheckCollisionPointRec(
			   raylib.GetMousePosition(),
			   raylib.Rectangle{160, 8, 150, 30},
		   ) &&
		   raylib.IsMouseButtonPressed(.LEFT) {
			state.current_tab = .Monitoring
		}

		// Dynamic Status Indicator rechts oben
		if state.is_loading_mon {
			raylib.DrawCircle(820, 22, 5, raylib.YELLOW)
		} else if state.has_stats {
			raylib.DrawCircle(820, 22, 5, raylib.GREEN)
		} else {
			raylib.DrawCircle(820, 22, 5, raylib.RED)
		}

		// --- TAB 1: CHAT VIEW ---
		if state.current_tab == .Chat {
			raylib.DrawRectangleLines(20, 65, 810, 400, raylib.DARKGRAY)
			draw_wrapped_text(state.agent_response, 35, 80, 780, 18, raylib.RAYWHITE)

			raylib.DrawRectangle(20, 480, 680, 42, raylib.Color{30, 30, 42, 255})
			raylib.DrawRectangleLines(20, 480, 680, 42, raylib.GRAY)

			c_input := strings.clone_to_cstring(string(state.input_buf[:state.input_len]))
			defer delete(c_input)
			raylib.DrawText(c_input, 30, 491, 18, raylib.GREEN)

			if state.is_loading_chat {
				raylib.DrawText("Warte...", 715, 491, 18, raylib.YELLOW)
			} else {
				raylib.DrawRectangle(710, 480, 120, 42, raylib.Color{40, 90, 50, 255})
				raylib.DrawText("Senden [↵]", 722, 492, 16, raylib.WHITE)
			}
		}

		// --- TAB 2: MONITORING VIEW ---
		if state.current_tab == .Monitoring {
			if !state.has_stats {
				c_conn_str := strings.clone_to_cstring(
					fmt.tprintf("Verbinde mit Server auf %s:9000...", state.config.server_ip),
				)
				defer delete(c_conn_str)
				raylib.DrawText(c_conn_str, 40, 80, 20, raylib.GRAY)
			} else {
				y_off: i32 = 75

				// 1. CPU
				cpu_str := strings.clone_to_cstring(
					fmt.tprintf("CPU Auslastung: %.1f%%", state.stats.cpu_usage_percent),
				)
				defer delete(cpu_str)
				raylib.DrawText(cpu_str, 40, y_off, 18, raylib.LIGHTGRAY)

				raylib.DrawRectangle(250, y_off, 300, 20, raylib.DARKGRAY)
				raylib.DrawRectangle(
					250,
					y_off,
					i32(300.0 * (state.stats.cpu_usage_percent / 100.0)),
					20,
					raylib.RED,
				)
				y_off += 45

				// 2. RAM
				ram_str := strings.clone_to_cstring(
					fmt.tprintf(
						"RAM Nutzung: %d MB / %d MB (%.1f%%)",
						state.stats.ram_used_mb,
						state.stats.ram_total_mb,
						state.stats.ram_usage_percent,
					),
				)
				defer delete(ram_str)
				raylib.DrawText(ram_str, 40, y_off, 18, raylib.LIGHTGRAY)

				raylib.DrawRectangle(420, y_off, 250, 20, raylib.DARKGRAY)
				raylib.DrawRectangle(
					420,
					y_off,
					i32(250.0 * (state.stats.ram_usage_percent / 100.0)),
					20,
					raylib.SKYBLUE,
				)
				y_off += 45

				// 3. Disk
				disk_str := strings.clone_to_cstring(
					fmt.tprintf(
						"Speicherplatz: %d GB frei von %d GB",
						state.stats.disk_free_gb,
						state.stats.disk_total_gb,
					),
				)
				defer delete(disk_str)
				raylib.DrawText(disk_str, 40, y_off, 18, raylib.LIGHTGRAY)
				y_off += 45

				// 4. Uptime
				uptime_str := strings.clone_to_cstring(
					fmt.tprintf(
						"Server Uptime: %s (%ds)",
						state.stats.uptime_formated,
						state.stats.uptime_seconds,
					),
				)
				defer delete(uptime_str)
				raylib.DrawText(uptime_str, 40, y_off, 18, raylib.GREEN)
				y_off += 45

				// 5. GPU
				gpu_str := strings.clone_to_cstring(
					fmt.tprintf("GPU Info: %s", state.stats.gpu_info),
				)
				defer delete(gpu_str)
				raylib.DrawText(gpu_str, 40, y_off, 18, raylib.GOLD)
			}
		}

		raylib.EndDrawing()

		if state.is_loading_chat && state.input_len > 0 {
			state.input_len = 0
		}
	}
}
