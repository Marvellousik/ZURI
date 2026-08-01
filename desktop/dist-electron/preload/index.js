import { contextBridge as e, ipcRenderer as t } from "electron";
//#region src/shared/ipc-channels.ts
var n = {
	DAEMON_GET_STATUS: "daemon:get-status",
	DAEMON_START: "daemon:start",
	DAEMON_STOP: "daemon:stop",
	DAEMON_RESTART: "daemon:restart",
	DAEMON_STATUS_CHANGED: "daemon:status-changed",
	LOG_MESSAGE: "log:message",
	APP_GET_INFO: "app:get-info"
};
//#endregion
//#region src/preload/index.ts
e.exposeInMainWorld("zuriAPI", {
	getDaemonStatus: () => t.invoke(n.DAEMON_GET_STATUS),
	startDaemon: () => t.invoke(n.DAEMON_START),
	stopDaemon: () => t.invoke(n.DAEMON_STOP),
	restartDaemon: () => t.invoke(n.DAEMON_RESTART),
	onDaemonStatusChanged: (e) => {
		let r = (t, n) => e(n);
		return t.on(n.DAEMON_STATUS_CHANGED, r), () => {
			t.removeListener(n.DAEMON_STATUS_CHANGED, r);
		};
	},
	logMessage: (e, r, i) => t.invoke(n.LOG_MESSAGE, {
		level: e,
		message: r,
		details: i
	}),
	getAppInfo: () => t.invoke(n.APP_GET_INFO)
});
//#endregion
export {};
