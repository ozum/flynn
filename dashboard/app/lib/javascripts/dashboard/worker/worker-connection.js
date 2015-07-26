import Connection from './connection';
Connection.addPort({
	postMessage: function () {
		postMessage.apply(null, arguments); // jshint ignore:line
	}
});
onmessage = function (e) { // jshint ignore:line
	Connection.handleMessage({
		data: e.data,
		respond: function (data) {
			postMessage(data); // jshint ignore:line
		}
	});
};
