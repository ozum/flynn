import StaticConfig from './static-config';
import Dispatcher from './dispatcher';

var worker = null;
var postMessage = null;
var WorkerAPI = {
	run: function () {
		if (window.SharedWorker) {
			worker = new SharedWorker(StaticConfig.ASSET_PATHS['dashboard-worker-shared.js']);
			postMessage = worker.port.postMessage.bind(worker.port);
			worker.port.onmessage = this.handleMessage.bind(this);
			worker.port.start();
		} else if (window.Worker) {
			worker = new Worker(StaticConfig.ASSET_PATHS['dashboard-worker.js']);
			postMessage = worker.postMessage.bind(worker);
			worker.onmessage = this.handleMessage.bind(this);
		} else {
			setTimeout(function () {
				window.alert('Browser compatibility error!');
			}, 0);
			throw new Error('Neither SharedWorker or Worker are supported!');
		}
		worker.onerror = function (e) {
			throw new Error(e.message + " (" + e.filename + ":" + e.lineno + ")");
		};
		this.postMessage({
			name: 'init',
			config: StaticConfig
		});
	},

	postMessage: function (data) {
		console.log('WorkerAPI.postMessage', data);
		postMessage(data);
	},

	handleMessage: function (e) {
		console.log('WorkerAPI.handleMessage', e.data);
		var event = e.data;
		switch (event.name) {
			case 'APP_EVENT':
				Dispatcher.handleAppEvent(event.data);
			break;
		}
	}
};
export default WorkerAPI;
