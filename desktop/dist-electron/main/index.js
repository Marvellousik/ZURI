import { BrowserWindow as e, Menu as t, app as n, ipcMain as r, shell as i } from "electron";
import a from "path";
import { spawn as o } from "child_process";
import s from "http";
import c from "fs";
import l from "events";
//#region src/main/window.ts
var u = class {
	mainWindow = null;
	createMainWindow() {
		let t = process.env.NODE_ENV === "development" || !n.isPackaged;
		if (this.mainWindow = new e({
			width: 1280,
			height: 830,
			minWidth: 1024,
			minHeight: 700,
			title: "Zuri — Engineering Memory System",
			backgroundColor: "#090d16",
			show: !1,
			webPreferences: {
				preload: a.join(__dirname, "../preload/index.js"),
				contextIsolation: !0,
				nodeIntegration: !1,
				sandbox: !0,
				webSecurity: !0
			}
		}), this.mainWindow.once("ready-to-show", () => {
			this.mainWindow?.show();
		}), t && process.env.VITE_DEV_SERVER_URL) console.log(`[WindowManager] Loading URL: ${process.env.VITE_DEV_SERVER_URL}`), this.mainWindow.loadURL(process.env.VITE_DEV_SERVER_URL), this.mainWindow.webContents.openDevTools({ mode: "detach" });
		else {
			let e = a.join(__dirname, "../../dist/index.html");
			console.log(`[WindowManager] Loading File: ${e}`), this.mainWindow.loadFile(e);
		}
		return this.mainWindow.on("closed", () => {
			this.mainWindow = null;
		}), this.mainWindow;
	}
	getMainWindow() {
		return this.mainWindow;
	}
}, d = class extends l {
	status = "STOPPED";
	process = null;
	port = 7331;
	host = "127.0.0.1";
	healthCheckInterval = null;
	startTime = null;
	lastError = void 0;
	constructor() {
		super();
		let e = process.env.ZURI_PORT;
		if (e) {
			let t = parseInt(e, 10);
			isNaN(t) || (this.port = t);
		}
		let t = process.env.ZURI_HOST;
		t && (this.host = t);
	}
	getDaemonInfo() {
		let e = this.startTime && this.status === "RUNNING" ? Math.floor((Date.now() - this.startTime) / 1e3) : void 0;
		return {
			status: this.status,
			port: this.port,
			host: this.host,
			pid: this.process?.pid,
			uptimeSeconds: e,
			error: this.lastError,
			lastCheckedAt: (/* @__PURE__ */ new Date()).toISOString()
		};
	}
	startMonitoring(e = 3e3) {
		this.healthCheckInterval && clearInterval(this.healthCheckInterval), this.checkHealth(), this.healthCheckInterval = setInterval(() => {
			this.checkHealth();
		}, e);
	}
	stopMonitoring() {
		this.healthCheckInterval &&= (clearInterval(this.healthCheckInterval), null);
	}
	async checkHealth() {
		return new Promise((e) => {
			let t = s.get(`http://${this.host}:${this.port}/health`, { timeout: 2e3 }, (t) => {
				t.statusCode === 200 ? (this.status !== "RUNNING" && (this.updateStatus("RUNNING"), this.startTime ||= Date.now()), e(!0)) : (this.status === "RUNNING" && this.updateStatus("ERROR", `Health check returned status code ${t.statusCode}`), e(!1));
			});
			t.on("error", (t) => {
				this.status === "RUNNING" ? this.updateStatus("STOPPED", t.message) : this.status, e(!1);
			}), t.end();
		});
	}
	async start() {
		if (this.status === "RUNNING" || this.status === "STARTING") return {
			success: !1,
			error: `Daemon is already ${this.status.toLowerCase()}`,
			daemonInfo: this.getDaemonInfo()
		};
		this.updateStatus("STARTING"), this.lastError = void 0;
		try {
			let e = process.env.NODE_ENV === "development" || !process.env.NODE_ENV, t = process.cwd(), n = c.existsSync(a.join(t, "go.mod")) ? t : a.resolve(t, ".."), r = process.platform === "win32" ? "zuri-daemon.exe" : "zuri-daemon", i = a.join(n, r);
			if (c.existsSync(i)) console.log(`[DaemonProcessManager] Spawning daemon executable from: ${i}`), this.process = o(i, [], {
				cwd: n,
				env: {
					...process.env,
					ZURI_PORT: String(this.port),
					ZURI_HOST: this.host
				},
				stdio: [
					"ignore",
					"pipe",
					"pipe"
				]
			});
			else if (e) console.log(`[DaemonProcessManager] Executable not found at ${i}. Spawning 'go run ./cmd/daemon' in ${n}...`), this.process = o("go", ["run", "./cmd/daemon"], {
				cwd: n,
				env: {
					...process.env,
					ZURI_PORT: String(this.port),
					ZURI_HOST: this.host
				},
				stdio: [
					"ignore",
					"pipe",
					"pipe"
				]
			});
			else throw Error(`Daemon executable not found at: ${i}`);
			return this.process && (this.process.stdout?.on("data", (e) => {
				console.log(`[Daemon STDOUT] ${e.toString().trim()}`);
			}), this.process.stderr?.on("data", (e) => {
				console.error(`[Daemon STDERR] ${e.toString().trim()}`);
			}), this.process.on("exit", (e, t) => {
				console.log(`[DaemonProcessManager] Process exited with code ${e}, signal ${t}`), this.process = null, this.startTime = null, this.status === "STOPPING" ? this.updateStatus("STOPPED") : this.updateStatus("STOPPED", e === 0 ? void 0 : `Exited with code ${e}`);
			}), this.process.on("error", (e) => {
				console.error("[DaemonProcessManager] Failed to start process:", e), this.process = null, this.startTime = null, this.updateStatus("ERROR", e.message);
			})), await this.pollUntilHealthy(1e4) ? (this.updateStatus("RUNNING"), this.startTime = Date.now(), {
				success: !0,
				message: "Daemon started successfully",
				daemonInfo: this.getDaemonInfo()
			}) : {
				success: !1,
				error: "Daemon process started but health check timed out",
				daemonInfo: this.getDaemonInfo()
			};
		} catch (e) {
			return this.updateStatus("ERROR", e.message), {
				success: !1,
				error: e.message,
				daemonInfo: this.getDaemonInfo()
			};
		}
	}
	async stop() {
		return this.status === "STOPPED" ? {
			success: !0,
			message: "Daemon is already stopped",
			daemonInfo: this.getDaemonInfo()
		} : (this.updateStatus("STOPPING"), this.process ? new Promise((e) => {
			let t = this.process, n = setTimeout(() => {
				t && !t.killed && (console.warn("[DaemonProcessManager] Graceful stop timed out. Killing process..."), t.kill("SIGKILL"));
			}, 5e3);
			t.once("exit", () => {
				clearTimeout(n), this.process = null, this.startTime = null, this.updateStatus("STOPPED"), e({
					success: !0,
					message: "Daemon stopped successfully",
					daemonInfo: this.getDaemonInfo()
				});
			}), t.kill("SIGTERM");
		}) : (this.updateStatus("STOPPED"), {
			success: !0,
			message: "Daemon process cleared",
			daemonInfo: this.getDaemonInfo()
		}));
	}
	async restart() {
		return console.log("[DaemonProcessManager] Restarting daemon..."), await this.stop(), await new Promise((e) => setTimeout(e, 1e3)), this.start();
	}
	updateStatus(e, t) {
		this.status = e, t !== void 0 && (this.lastError = t), this.emit("status-changed", this.getDaemonInfo());
	}
	async pollUntilHealthy(e) {
		let t = Date.now();
		for (; Date.now() - t < e;) {
			if (await this.checkHealth()) return !0;
			await new Promise((e) => setTimeout(e, 500));
		}
		return !1;
	}
}, f = {
	DAEMON_GET_STATUS: "daemon:get-status",
	DAEMON_START: "daemon:start",
	DAEMON_STOP: "daemon:stop",
	DAEMON_RESTART: "daemon:restart",
	DAEMON_STATUS_CHANGED: "daemon:status-changed",
	LOG_MESSAGE: "log:message",
	APP_GET_INFO: "app:get-info"
};
//#endregion
//#region src/ipc/main-handlers.ts
function p(e, t) {
	r.handle(f.DAEMON_GET_STATUS, async () => e.getDaemonInfo()), r.handle(f.DAEMON_START, async () => await e.start()), r.handle(f.DAEMON_STOP, async () => await e.stop()), r.handle(f.DAEMON_RESTART, async () => await e.restart()), r.handle(f.LOG_MESSAGE, async (e, { level: t, message: n, details: r }) => {
		(/* @__PURE__ */ new Date()).toISOString();
		let i = `[Renderer Log] [${t.toUpperCase()}] ${n}`;
		t === "error" ? console.error(i, r || "") : t === "warn" ? console.warn(i, r || "") : console.log(i, r || "");
	}), r.handle(f.APP_GET_INFO, async () => ({
		name: n.getName(),
		version: n.getVersion(),
		platform: process.platform,
		arch: process.arch,
		isDev: process.env.NODE_ENV === "development" || !n.isPackaged
	})), e.on("status-changed", (e) => {
		let n = t();
		n && !n.isDestroyed() && n.webContents.send(f.DAEMON_STATUS_CHANGED, e);
	});
}
//#endregion
//#region src/main/menu.ts
function m(e) {
	let r = process.platform === "darwin", a = [
		...r ? [{
			label: n.name,
			submenu: [
				{ role: "about" },
				{ type: "separator" },
				{ role: "services" },
				{ type: "separator" },
				{ role: "hide" },
				{ role: "hideOthers" },
				{ role: "unhide" },
				{ type: "separator" },
				{ role: "quit" }
			]
		}] : [],
		{
			label: "File",
			submenu: [r ? { role: "close" } : { role: "quit" }]
		},
		{
			label: "Daemon",
			submenu: [
				{
					label: "Start Daemon",
					accelerator: "CmdOrCtrl+Shift+S",
					click: () => e.start()
				},
				{
					label: "Stop Daemon",
					accelerator: "CmdOrCtrl+Shift+X",
					click: () => e.stop()
				},
				{
					label: "Restart Daemon",
					accelerator: "CmdOrCtrl+Shift+R",
					click: () => e.restart()
				},
				{ type: "separator" },
				{
					label: "Check Daemon Health",
					click: () => e.checkHealth()
				}
			]
		},
		{
			label: "View",
			submenu: [
				{ role: "reload" },
				{ role: "forceReload" },
				{ role: "toggleDevTools" },
				{ type: "separator" },
				{ role: "resetZoom" },
				{ role: "zoomIn" },
				{ role: "zoomOut" },
				{ type: "separator" },
				{ role: "togglefullscreen" }
			]
		},
		{
			label: "Help",
			submenu: [{
				label: "Zuri Spec & Documentation",
				click: async () => {
					await i.openExternal("https://github.com/Marvellousik/ZURI");
				}
			}]
		}
	], o = t.buildFromTemplate(a);
	t.setApplicationMenu(o);
}
//#endregion
//#region src/main/index.ts
var h = null, g = null;
async function _() {
	console.log("[Main Process] Initializing Zuri Desktop Application Shell..."), h = new u(), g = new d(), m(g), p(g, () => h?.getMainWindow() || null), h.createMainWindow(), g.startMonitoring(3e3), n.on("activate", () => {
		e.getAllWindows().length === 0 && h && h.createMainWindow();
	});
}
n.whenReady().then(_), n.on("window-all-closed", async () => {
	console.log("[Main Process] All windows closed."), g && (g.stopMonitoring(), await g.stop()), process.platform !== "darwin" && n.quit();
}), n.on("before-quit", async () => {
	g && (g.stopMonitoring(), await g.stop());
});
//#endregion
export {};
