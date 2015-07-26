import Connection from './connection';
onconnect = function (e) { // jshint ignore:line
	var port = e.ports[0];
	Connection.addPort(port);
	port.onmessage = function (e) {
		var message = {
			data: e.data,
			respond: function (data) {
				port.postMessage(data);
			}
		};
		Connection.handleMessage(message);
	};
	port.start();
};
