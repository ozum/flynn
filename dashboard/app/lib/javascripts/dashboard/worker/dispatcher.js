import BaseDispatcher from 'marbles/dispatcher';
import { extend } from 'marbles/utils';

var Dispatcher = extend({
	handleAppEvent: function (event) {
		this.dispatch({
			name: 'APP_EVENT',
			data: event
		});
	}
}, BaseDispatcher);

export default Dispatcher;
