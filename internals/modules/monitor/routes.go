package monitor

import "github.com/go-chi/chi/v5"

func Routes(h *Handler) chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.CreateMonitor)
	r.Get("/", h.GetAllMonitors)
	r.Get("/{monitorID}", h.GetMonitor)
	r.Patch("/{monitorID}", h.UpdateMonitorStatus)

	return r
}


/*
- POST: /monitors  -> create monitor
	req auth : true
	body : CreateMonitorRequest
	resp : monitorID

- GET: /monitors?offset={}&limit={}   -> get all monitors of a user
	req auth : true
	body : nil
	resp : GetAllMonitorsResponse

- GET: /monitors/{monitorID} -> get details of a monitor
	req auth : true
	body : nil
	resp : GetMonitorResponse

- PATCH: /monitors/{monitorID} -> update monitor status 
	req auth : true
	body : UpdateMonitorStatusRequest
	resp : ok / error
*/