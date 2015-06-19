package main

import (
	"net/http"

	"github.com/flynn/flynn/Godeps/_workspace/src/github.com/flynn/go-sql"
	"github.com/flynn/flynn/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/flynn/flynn/controller/schema"
	ct "github.com/flynn/flynn/controller/types"
	"github.com/flynn/flynn/pkg/httphelper"
	"github.com/flynn/flynn/pkg/postgres"
	routerc "github.com/flynn/flynn/router/client"
	"github.com/flynn/flynn/router/types"
)

func wrapDBExec(dbExec func(string, ...interface{}) error) func(string, ...interface{}) (sql.Result, error) {
	return func(q string, args ...interface{}) (sql.Result, error) {
		return nil, dbExec(q, args...)
	}
}

func createRoute(db *postgres.DB, rc routerc.Client, appID string, route *router.Route) error {
	route.ParentRef = routeParentRef(appID)
	if err := schema.Validate(route); err != nil {
		return err
	}
	if err := rc.CreateRoute(route); err != nil {
		return err
	}
	return createEvent(wrapDBExec(db.Exec), appID, route.ID, ct.EventTypeRoute, route)
}

func (c *controllerAPI) CreateRoute(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	var route router.Route
	if err := httphelper.DecodeJSON(req, &route); err != nil {
		respondWithError(w, err)
		return
	}

	if err := createRoute(c.appRepo.db, c.routerc, c.getApp(ctx).ID, &route); err != nil {
		respondWithError(w, err)
		return
	}

	httphelper.JSON(w, 200, &route)
}

func (c *controllerAPI) GetRoute(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	route, err := c.getRoute(ctx)
	if err != nil {
		respondWithError(w, err)
		return
	}

	httphelper.JSON(w, 200, route)
}

func (c *controllerAPI) GetRouteList(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	routes, err := c.routerc.ListRoutes(routeParentRef(c.getApp(ctx).ID))
	if err != nil {
		respondWithError(w, err)
		return
	}
	httphelper.JSON(w, 200, routes)
}

func (c *controllerAPI) DeleteRoute(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	route, err := c.getRoute(ctx)
	if err != nil {
		respondWithError(w, err)
		return
	}

	err = c.routerc.DeleteRoute(route.Type, route.ID)
	if err == routerc.ErrNotFound {
		err = ErrNotFound
	}
	if err != nil {
		respondWithError(w, err)
		return
	}
	appID := c.getApp(ctx).ID
	if err := createEvent(wrapDBExec(c.appRepo.db.Exec), appID, route.ID, ct.EventTypeRouteDeletion, route); err != nil {
		respondWithError(w, err)
		return
	}
	w.WriteHeader(200)
}
