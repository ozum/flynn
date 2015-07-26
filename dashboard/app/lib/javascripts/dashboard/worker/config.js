import { extend } from 'marbles/utils';
import HTTP from 'marbles/http';
import WithCredentialsMiddleware from 'marbles/http/middleware/with_credentials';
import SerializeJSONMiddleware from 'marbles/http/middleware/serialize_json';
import Dispatcher from './dispatcher';

var Config = {
	waitForRouteHandler: Promise.resolve(),
	client: null,
	githubClient: null,
	fetchInProgress: false
};
var fetchedProperties = [];

Config.extend = function (config) {
	extend(Config, config);
};

var appEvents = [];
Config.getAppEvents = function () {
	return appEvents;
};

Config.fetch = function () {
	Config.fetchInProgress = true;
	var resolve, reject;
	var promise = new Promise(function (rs, rj) {
		resolve = function () {
			Config.fetchInProgress = false;
			rs();
		};
		reject = function () {
			Config.fetchInProgress = false;
			rj();
		};
	});

	var handleFailure = function (res, xhr) {
		Config.err = new Error("SERVICE_UNAVAILABLE");
		
		appEvents = [{
			name: "SERVICE_UNAVAILABLE",
			status: xhr.status
		}];

		reject(Config.err);
	};

	var handleSuccess = function (res) {
		Config.err = null;

		// clear all fetched properties from Config
		fetchedProperties.forEach(function (k) {
			delete Config[k];
		});
		fetchedProperties = [];

		// add fetched properties to Config
		for (var k in res) {
			if (res.hasOwnProperty(k)) {
				fetchedProperties.push(k);
				Config[k] = res[k];
			}
		}

		// make all endpoints absolute URLs
		var endpoints = Config.endpoints;
		for (k in endpoints) {
			if (endpoints.hasOwnProperty(k) && endpoints[k][0] === "/") {
				endpoints[k] = Config.API_SERVER + endpoints[k];
			}
		}

		var authenticated = res.hasOwnProperty("user");
		var authenticatedChanged = false;
		if (authenticated !== Config.authenticated) {
			authenticatedChanged = true;
			Config.authenticated = authenticated;
		}

		var githubAuthenticated = !!(res.user && res.user.auths && res.user.auths.hasOwnProperty("github"));
		var githubAuthenticatedChanged = false;
		if (githubAuthenticated !== Config.githubAuthenticated) {
			githubAuthenticatedChanged = true;
			Config.githubAuthenticated = githubAuthenticated;
		}

		appEvents = [{
			name: 'CONFIG_READY'
		}];

		if (authenticatedChanged) {
			appEvents.push({
				name: "AUTH_CHANGE",
				authenticated: authenticated
			});
		}

		if (githubAuthenticatedChanged) {
			appEvents.push({
				name: "GITHUB_AUTH_CHANGE",
				authenticated: githubAuthenticated
			});
		}

		resolve(Config);
	};

	HTTP({
		method: 'GET',
		url: Config.API_SERVER.replace(/^https?/, Config.HTTPS ? 'https' : 'http') + "/config",
		middleware: [
			WithCredentialsMiddleware,
			SerializeJSONMiddleware
		],
		callback: function (res, xhr) {
			if (xhr.status !== 200 || !String(xhr.getResponseHeader('Content-Type')).match(/application\/json/)) {
				handleFailure(res, xhr);
			} else {
				handleSuccess(res, xhr);
			}
		}
	});

	var dispatchAppEvents = function () {
		appEvents.forEach(function (event) {
			Dispatcher.handleAppEvent(event);
		});
	};

	return promise.then(dispatchAppEvents, dispatchAppEvents);
};

Config.setGithubToken = function (token) {
	Config.user.auths.github = { access_token: token };
	Dispatcher.handleAppEvent({
		name: "GITHUB_AUTH_CHANGE",
		authenticated: true
	});
};

export default Config;
