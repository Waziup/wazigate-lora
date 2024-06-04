package app

import (
	"context"
	"encoding/json"
	"net/http"

	asAPI "github.com/chirpstack/chirpstack/api/go/v4/api"
)

func serveAPI(resp http.ResponseWriter, req *http.Request) {

	switch req.URL.Path {
	case "/randomDevAddr":
		if req.Method == http.MethodPost {
			decoder := json.NewDecoder(req.Body)
			var devEUI string
			if err := decoder.Decode(&devEUI); err != nil {
				serveError(resp, err)
				return
			}
			deviceService := asAPI.NewDeviceServiceClient(chirpstack)
			r, err := deviceService.GetRandomDevAddr(context.Background(), &asAPI.GetRandomDevAddrRequest{
				DevEui: devEUI,
			})
			if err != nil {
				serveError(resp, err)
				return
			}
			serveJSON(resp, r.DevAddr)
			return
		}
	case "/profiles":
		switch req.Method {
		case http.MethodGet:
			deviceProfileService := asAPI.NewDeviceProfileServiceClient(chirpstack)
			r, err := deviceProfileService.List(context.Background(), &asAPI.ListDeviceProfilesRequest{
				Limit:    1000,
				TenantId: Config.Tenant.Id,
			})
			if err != nil {
				serveError(resp, err)
				return
			}
			serveJSON(resp, r.Result)
			return
		case http.MethodPost:
			decoder := json.NewDecoder(req.Body)
			var deviceProfile asAPI.DeviceProfile
			if err := decoder.Decode(&deviceProfile); err != nil {
				serveError(resp, err)
				return
			}
			deviceProfileService := asAPI.NewDeviceProfileServiceClient(chirpstack)
			deviceProfile.TenantId = Config.Tenant.Id
			if deviceProfile.Id == "" {
				r, err := deviceProfileService.Create(context.Background(), &asAPI.CreateDeviceProfileRequest{
					DeviceProfile: &deviceProfile,
				})
				if err != nil {
					serveError(resp, err)
					return
				}
				serveJSON(resp, r.Id)
				return
			}
			_, err := deviceProfileService.Update(context.Background(), &asAPI.UpdateDeviceProfileRequest{
				DeviceProfile: &deviceProfile,
			})
			if err != nil {
				serveError(resp, err)
				return
			}
			return
		}
	}

	serveStatic(resp, req)
}

func serveError(resp http.ResponseWriter, err error) {
	resp.Header().Set("Content-Type", "text/plain")
	resp.WriteHeader(http.StatusInternalServerError)
	resp.Write([]byte(err.Error()))
}

func serveJSON(resp http.ResponseWriter, data interface{}) {
	body, err := json.Marshal(data)
	if err != nil {
		serveError(resp, err)
		return
	}
	resp.Header().Set("Content-Type", "application/json; charset=utf-8")
	resp.Write(body)
}
