import Dispatcher from './dispatcher';
import Config from './config';

var ports = [];
var configInitialized = false;
var Connection = {
	handleMessage: function (m) {
		switch (m.data.name) {
			case 'init':
				if ( !configInitialized ) {
					configInitialized = true;
					Dispatcher.dispatch({
						name: 'STATIC_CONFIG',
						data: m.data.config
					});
				}
				if ( !Config.fetchInProgress ) {
					Config.getAppEvents().forEach(function (event) {
						m.respond({
							name: 'APP_EVENT',
							data: event
						});
					});
				}
			break;

			default:
				m.respond({
					hello: 'from worker'
				});
		}
	},

	postMessage: function (data) {
		ports.forEach(function (p) {
			try {
				p.postMessage(data);
			} catch (e) {}
		});
	},

	addPort: function (port) {
		ports.push(port);
	},

	handleEvent: function (event) {
		if (event.name !== 'APP_EVENT') {
			return;
		}
		this.postMessage({
			name: 'APP_EVENT',
			data: event.data
		});
	}
};

Dispatcher.register(Connection.handleEvent.bind(Connection));

export default Connection;
