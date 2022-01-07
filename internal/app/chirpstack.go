package app

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"

	asAPI "github.com/brocaar/chirpstack-api/go/v3/as/external/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var chirpstack *grpc.ClientConn

const chirpstackTokenRefreshInterval = 5 * time.Minute

func connectToChirpStack() error {
	var err error
	chirpstack, err = grpc.Dial("waziup.wazigate-lora.chirpstack-application-server:8080",
		grpc.WithBlock(),
		grpc.WithPerRPCCredentials(jwtCredentials),
		grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("grpc: can not dial: %v", err)
	}

	internalClient := asAPI.NewInternalServiceClient(chirpstack)
	resp, err := internalClient.Login(context.Background(), &Config.Login)
	if err != nil {
		return fmt.Errorf("grpc: can not login: %v", err)
	}

	jwtCredentials.SetToken(resp.Jwt)
	return nil
}

func refreshChirpstackToken() {
	for {
		time.Sleep(chirpstackTokenRefreshInterval)
		internalClient := asAPI.NewInternalServiceClient(chirpstack)
		resp, err := internalClient.Login(context.Background(), &Config.Login)
		if err != nil {
			log.Fatalf("grpc: can not refresh token: %v", err)
		}
		jwtCredentials.SetToken(resp.Jwt)
	}
}

func InitChirpstack() error {

	if err := connectToChirpStack(); err != nil {
		return err
	}

	go refreshChirpstackToken()
	log.Println("--- Init ChirpStack")

	dirty := false

	defer func() {
		if dirty {
			fmt.Println("The ChirpStack data changed, see 'chirpstack.json'.")
			if err := WriteConfig(); err != nil {
				panic(fmt.Errorf("can not write 'chirpstack.json': %v", err))
			}
		} else {
			fmt.Println("The ChirpStack data has not changed.")
		}
	}()

	ctx := context.Background()
	{
		asOrganizationService := asAPI.NewOrganizationServiceClient(chirpstack)
		resp, err := asOrganizationService.Get(ctx, &asAPI.GetOrganizationRequest{
			Id: Config.Organization.Id,
		})
		if err != nil {
			if status.Code(err) == codes.NotFound {
				log.Printf("Organization id %d does not exist !?", Config.Organization.Id)

				Config.Organization.Id = 0
				Config.Organization.Name = "wazigate_" + strconv.Itoa(rand.Int())
				Config.Organization.DisplayName = "Local Wazigate"
				resp, err := asOrganizationService.Create(ctx, &asAPI.CreateOrganizationRequest{
					Organization: &Config.Organization,
				})
				if err != nil {
					return fmt.Errorf("grpc: can not create organization: %v", err)
				}
				Config.Organization.Id = resp.Id
				dirty = true
				log.Printf("Organization has been recreated. ID: %v", Config.Organization.Id)
			} else {
				return fmt.Errorf("grpc: can not get Organization: %v", err)
			}
		} else {
			log.Printf("Organization %q OK.", resp.Organization.Name)
		}
	}
	{
		asNetworkServerService := asAPI.NewNetworkServerServiceClient(chirpstack)
		resp, err := asNetworkServerService.List(ctx, &asAPI.ListNetworkServerRequest{
			Limit:          1000,
			OrganizationId: Config.Organization.Id,
		})
		if err != nil {
			return fmt.Errorf("grpc: can not list network-servers: %v", err)
		}
		for _, ns := range resp.Result {
			if ns.Server == Config.NetworkServer.Server {
				if ns.Id != Config.NetworkServer.Id {
					log.Printf("A network-server with the same configuration exists? !?. ID: %v <> %v", Config.NetworkServer.Id, ns.Id)
					Config.NetworkServer.Id = ns.Id
					dirty = true
				}
				break
			}
		}

		if Config.NetworkServer.Id == 0 {
			resp, err := asNetworkServerService.Create(ctx, &asAPI.CreateNetworkServerRequest{
				NetworkServer: &Config.NetworkServer,
			})
			if err != nil {
				return fmt.Errorf("grpc: can not create network-server: %v", err)
			}
			Config.NetworkServer.Id = resp.Id
			log.Printf("Network-server has been created. ID: %v", Config.NetworkServer.Id)
			dirty = true
		} else {
			resp, err := asNetworkServerService.Get(ctx, &asAPI.GetNetworkServerRequest{
				Id: Config.NetworkServer.Id,
			})
			if err != nil {
				if status.Code(err) == codes.NotFound {
					log.Printf("Network-server id %d does not exist !?", Config.NetworkServer.Id)
					resp, err := asNetworkServerService.Create(ctx, &asAPI.CreateNetworkServerRequest{
						NetworkServer: &Config.NetworkServer,
					})
					if err != nil {
						return fmt.Errorf("grpc: can not create network-server: %v", err)
					}
					Config.NetworkServer.Id = resp.Id
					log.Printf("Network-server has been recreated. ID: %v", Config.NetworkServer.Id)
					dirty = true
				} else {
					return fmt.Errorf("grpc: can not get network-server: %v", err)
				}
			} else {
				log.Printf("Network-server %q OK.", resp.NetworkServer.Name)
			}
		}
	}
	{
		asServiceProfileService := asAPI.NewServiceProfileServiceClient(chirpstack)
		Config.ServiceProfile.NetworkServerId = Config.NetworkServer.Id
		Config.ServiceProfile.OrganizationId = Config.Organization.Id

		resp, err := asServiceProfileService.List(ctx, &asAPI.ListServiceProfileRequest{
			Limit:          1000,
			OrganizationId: Config.Organization.Id,
		})
		if err != nil {
			return fmt.Errorf("grpc: can not list service-profile: %v", err)
		}
		for _, sp := range resp.Result {
			if sp.NetworkServerId == Config.NetworkServer.Id {
				if sp.Id != Config.ServiceProfile.Id {
					log.Printf("A  service-profile with the same configuration exists? !?. ID: %v <> %v", Config.ServiceProfile.Id, sp.Id)
					Config.ServiceProfile.Id = sp.Id
					dirty = true
				}
				break
			}
		}

		if Config.ServiceProfile.Id == "" {
			resp, err := asServiceProfileService.Create(ctx, &asAPI.CreateServiceProfileRequest{
				ServiceProfile: &Config.ServiceProfile,
			})
			if err != nil {
				return fmt.Errorf("grpc: can not create service-profile: %v", err)
			}
			Config.ServiceProfile.Id = resp.Id
			log.Printf("Service-profile has been created. ID: %v", Config.ServiceProfile.Id)
			dirty = true
		} else {
			resp, err := asServiceProfileService.Get(ctx, &asAPI.GetServiceProfileRequest{
				Id: Config.ServiceProfile.Id,
			})
			if err != nil {
				if status.Code(err) == codes.NotFound {
					log.Printf("Service-profile id '%v' does not exist !?", Config.ServiceProfile.Id)

					resp, err := asServiceProfileService.Create(ctx, &asAPI.CreateServiceProfileRequest{
						ServiceProfile: &Config.ServiceProfile,
					})
					if err != nil {
						return fmt.Errorf("grpc: can not create service-profile: %v", err)
					}
					Config.ServiceProfile.Id = resp.Id
					log.Printf("Service-profile has been recreated. ID: %v", Config.ServiceProfile.Id)
					dirty = true
				} else {
					return fmt.Errorf("grpc: can not get service-profile: %v", err)
				}
			} else {
				log.Printf("Service-profile %q OK.", resp.ServiceProfile.Name)
			}
		}
	}
	{
		asGatewayService := asAPI.NewGatewayServiceClient(chirpstack)
		Config.Gateway.OrganizationId = Config.Organization.Id
		resp, err := asGatewayService.Get(ctx, &asAPI.GetGatewayRequest{
			Id: Config.Gateway.Id,
		})
		if err != nil {
			// log.Printf("\n\n\t-->\tGholi: %q\n\n", err)
			if status.Code(err) == codes.NotFound {
				Config.Gateway.NetworkServerId = Config.NetworkServer.Id
				Config.Gateway.OrganizationId = Config.Organization.Id
				_, err = asGatewayService.Create(ctx, &asAPI.CreateGatewayRequest{
					Gateway: &Config.Gateway,
				})
				if err != nil {
					return fmt.Errorf("grpc: can not create gateway: %v", err)
				}
				log.Printf("Gateway has been created. ID: %v", Config.Gateway.Id)
			} else {
				return fmt.Errorf("grpc: can not get gateway: %v", err)
			}
		} else {
			log.Printf("Gateway %q OK.", resp.Gateway.Name)
		}
	}
	{
		asApplicationService := asAPI.NewApplicationServiceClient(chirpstack)
		Config.Application.OrganizationId = Config.Organization.Id
		Config.Application.ServiceProfileId = Config.ServiceProfile.Id
		if Config.Application.Id == 0 {
			resp, err := asApplicationService.Create(ctx, &asAPI.CreateApplicationRequest{
				Application: &Config.Application,
			})
			if err != nil {
				return fmt.Errorf("grpc: can not create application: %v", err)
			}
			Config.Application.Id = resp.Id
			log.Printf("Application has been created. ID: %v", Config.Application.Id)
			dirty = true
		} else {
			resp, err := asApplicationService.Get(ctx, &asAPI.GetApplicationRequest{
				Id: Config.Application.Id,
			})
			if err != nil {
				if status.Code(err) == codes.NotFound {
					log.Printf("Application id %v does not exist !?", Config.Application.Id)
					Config.Application.ServiceProfileId = Config.ServiceProfile.Id
					resp, err := asApplicationService.Create(ctx, &asAPI.CreateApplicationRequest{
						Application: &Config.Application,
					})
					if err != nil {
						return fmt.Errorf("grpc: can not create application: %v", err)
					}
					Config.Application.Id = resp.Id
					log.Printf("Application has been recreated. ID: %v", Config.Application.Id)
					dirty = true
				} else {
					return fmt.Errorf("grpc: can not get application: %v", err)
				}
			} else {
				log.Printf("Application %q OK.", resp.Application.Name)
			}
		}
	}
	{
		asDeviceProfileService := asAPI.NewDeviceProfileServiceClient(chirpstack)
		for i, deviceProfile := range Config.DeviceProfiles {
			deviceProfile.OrganizationId = Config.Organization.Id
			deviceProfile.NetworkServerId = Config.NetworkServer.Id
			if deviceProfile.Id == "" {
				resp, err := asDeviceProfileService.Create(ctx, &asAPI.CreateDeviceProfileRequest{
					DeviceProfile: &deviceProfile,
				})
				if err != nil {
					return fmt.Errorf("err grpc: can not create device-profile: %v", err)
				}
				deviceProfile.Id = resp.Id
				Config.DeviceProfiles[i] = deviceProfile
				log.Printf("Device-profile has been created. ID: %v", deviceProfile.Id)
				dirty = true
			} else {
				resp, err := asDeviceProfileService.Get(ctx, &asAPI.GetDeviceProfileRequest{
					Id: deviceProfile.Id,
				})
				if err != nil {
					if status.Code(err) == codes.NotFound {
						log.Printf("Device-profile id %q does not exist!", deviceProfile.Id)
						resp, err := asDeviceProfileService.Create(ctx, &asAPI.CreateDeviceProfileRequest{
							DeviceProfile: &deviceProfile,
						})
						if err != nil {
							return fmt.Errorf("grpc: can not create device-profile: %v", err)
						}
						deviceProfile.Id = resp.Id
						Config.DeviceProfiles[i] = deviceProfile
						log.Printf("Device-profile has been recreated. ID: %v", deviceProfile.Id)
						dirty = true
					} else {
						return fmt.Errorf("grpc: can not get device-profile: %v", err)
					}
				} else {
					log.Printf("Device-profile %q OK.", resp.DeviceProfile.Name)
				}
			}
		}
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////

func setDeviceProfileWaziDev(devEUI string, id string) error {
	ctx := context.Background()
	deviceProfileId := Config.DeviceProfiles[0].Id
	deviceClient := asAPI.NewDeviceServiceClient(chirpstack)
	resp, err := deviceClient.Get(ctx, &asAPI.GetDeviceRequest{
		DevEui: devEUI,
	})
	if status.Code(err) == codes.NotFound {
		_, err := deviceClient.Create(ctx, &asAPI.CreateDeviceRequest{
			Device: &asAPI.Device{
				DevEui:          devEUI,
				Name:            devEUI,
				Description:     fmt.Sprintf("Automatically created for Waziup device %q.\nDO NOT DELETE!", id),
				DeviceProfileId: deviceProfileId,
				ApplicationId:   Config.Application.Id,
				SkipFCntCheck:   true,
			},
		})
		if err == nil {
			log.Println("Creating Chirpstack device ... OK")
		} else {
			log.Printf("Err Can not create Chirpstack device: %v", err)
		}
		return err
	}
	if err != nil {
		log.Printf("Err Can not read Chirpstack device: %v", err)
		return err
	}
	if resp.Device.DeviceProfileId == deviceProfileId {
		return nil
	}
	_, err = deviceClient.Update(ctx, &asAPI.UpdateDeviceRequest{
		Device: &asAPI.Device{
			DevEui:          devEUI,
			ApplicationId:   Config.Application.Id,
			DeviceProfileId: deviceProfileId,
			Name:            resp.Device.Name,
			Description:     resp.Device.Description,
			SkipFCntCheck:   true,
		},
	})
	if err == nil {
		log.Println("Updating Chirpstack device ... OK")
	} else {
		log.Printf("Err Can not update Chirpstack device: %v", err)
	}
	return err
}

func setWaziDevActivation(devEUI string, devAddr string, nwkSEncKey string, appSKey string) error {
	ctx := context.Background()
	deviceClient := asAPI.NewDeviceServiceClient(chirpstack)
	r, err := deviceClient.GetActivation(ctx, &asAPI.GetDeviceActivationRequest{
		DevEui: devEUI,
	})
	if status.Code(err) == codes.NotFound {
		_, err = deviceClient.Activate(ctx, &asAPI.ActivateDeviceRequest{
			DeviceActivation: &asAPI.DeviceActivation{
				DevEui:      devEUI,
				DevAddr:     devAddr,
				AppSKey:     appSKey,
				NwkSEncKey:  nwkSEncKey,
				SNwkSIntKey: nwkSEncKey,
				FNwkSIntKey: nwkSEncKey,
				FCntUp:      0,
				NFCntDown:   0,
				AFCntDown:   0,
			},
		})
		if err == nil {
			log.Println("Activating Chirpstack device ... OK")
		} else {
			log.Printf("Err Can not activate Chirpstack device: %v", err)
		}
		return err
	}
	if err != nil {
		log.Printf("Err Can not get Chirpstack device activation: %v", err)
		return err
	}
	if r.DeviceActivation.DevEui != devEUI ||
		r.DeviceActivation.DevAddr != devAddr ||
		r.DeviceActivation.AppSKey != appSKey ||
		r.DeviceActivation.NwkSEncKey != nwkSEncKey {

		_, err = deviceClient.Activate(ctx, &asAPI.ActivateDeviceRequest{
			DeviceActivation: &asAPI.DeviceActivation{
				DevEui:      devEUI,
				DevAddr:     devAddr,
				AppSKey:     appSKey,
				NwkSEncKey:  nwkSEncKey,
				SNwkSIntKey: nwkSEncKey,
				FNwkSIntKey: nwkSEncKey,
				FCntUp:      0,
				NFCntDown:   0,
				AFCntDown:   0,
			},
		})
		if err == nil {
			log.Println("Reactivating Chirpstack device ... OK")
		} else {
			log.Printf("Err Can reactivate Chirpstack device: %v", err)
		}
		return err
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////

var jwtCredentials = &JWTCredentials{}

// JWTCredentials provides JWT credentials for gRPC
type JWTCredentials struct {
	token string
}

// GetRequestMetadata returns the meta-data for a request.
func (j *JWTCredentials) GetRequestMetadata(ctx context.Context, url ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": j.token,
	}, nil
}

// RequireTransportSecurity ...
func (j *JWTCredentials) RequireTransportSecurity() bool {
	return false
}

// SetToken sets the JWT token.
func (j *JWTCredentials) SetToken(token string) {
	j.token = token
}
