import { extend } from 'marbles/utils';
import Config from './config';
import Dispatcher from './dispatcher';

var DashboardWorker = function () {
};

extend(DashboardWorker.prototype, {
	run: function () {
		Dispatcher.register(this.handleEvent.bind(this));
	},

	handleEvent: function (event) {
		switch (event.name) {
			case 'STATIC_CONFIG':
				Config.extend(event.data);
				Config.fetch();
			break;
		}
	}
});

(new DashboardWorker()).run();
