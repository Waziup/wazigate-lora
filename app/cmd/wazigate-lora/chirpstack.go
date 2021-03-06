package main

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

const chirpstackTokenRefeshInterval = 5 * time.Minute

func connectToChirpStack() error {
	var err error
	chirpstack, err = grpc.Dial("chirpstack-application-server:8080",
		grpc.WithBlock(),
		grpc.WithPerRPCCredentials(jwtCredentials),
		grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("grpc: can not dial: %v", err)
	}

	internalClient := asAPI.NewInternalServiceClient(chirpstack)
	resp, err := internalClient.Login(context.Background(), &config.Login)
	if err != nil {
		return fmt.Errorf("grpc: can not login: %v", err)
	}

	jwtCredentials.SetToken(resp.Jwt)
	return nil
}

func refreshChirpstackToken() {
	for {
		time.Sleep(chirpstackTokenRefeshInterval)
		internalClient := asAPI.NewInternalServiceClient(chirpstack)
		resp, err := internalClient.Login(context.Background(), &config.Login)
		if err != nil {
			log.Fatalf("grpc: can not refresh token: %v", err)
		}
		jwtCredentials.SetToken(resp.Jwt)
	}
}

func initChirpstack() error {

	if err := connectToChirpStack(); err != nil {
		return nil
	}

	go refreshChirpstackToken()
	log.Println("--- Init ChirpStack")

	dirty := false

	defer func() {
		if dirty {
			fmt.Println("The ChirpStack data changed, see 'chirpstack.json'.")
			if err := writeConfig(); err != nil {
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
			Id: config.Organization.Id,
		})
		if err != nil {
			if status.Code(err) == codes.NotFound {
				log.Printf("Organization id %d does not exist !?", config.Organization.Id)

				config.Organization.Id = 0
				config.Organization.Name = "wazigate_" + strconv.Itoa(rand.Int())
				config.Organization.DisplayName = "Local Wazigate"
				resp, err := asOrganizationService.Create(ctx, &asAPI.CreateOrganizationRequest{
					Organization: &config.Organization,
				})
				if err != nil {
					return fmt.Errorf("grpc: can not create organization: %v", err)
				}
				config.Organization.Id = resp.Id
				dirty = true
				log.Printf("Organization has been recreated. ID: %v", config.Organization.Id)
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
			OrganizationId: config.Organization.Id,
		})
		if err != nil {
			return fmt.Errorf("grpc: can not list network-servers: %v", err)
		}
		for _, ns := range resp.Result {
			if ns.Server == config.NetworkServer.Server {
				if ns.Id != config.NetworkServer.Id {
					log.Printf("A network-server with the same configuration exists? !?. ID: %v <> %v", config.NetworkServer.Id, ns.Id)
					config.NetworkServer.Id = ns.Id
					dirty = true
				}
				break
			}
		}

		if config.NetworkServer.Id == 0 {
			resp, err := asNetworkServerService.Create(ctx, &asAPI.CreateNetworkServerRequest{
				NetworkServer: &config.NetworkServer,
			})
			if err != nil {
				return fmt.Errorf("grpc: can not create network-server: %v", err)
			}
			config.NetworkServer.Id = resp.Id
			log.Printf("Network-server has been created. ID: %v", config.NetworkServer.Id)
			dirty = true
		} else {
			resp, err := asNetworkServerService.Get(ctx, &asAPI.GetNetworkServerRequest{
				Id: config.NetworkServer.Id,
			})
			if err != nil {
				if status.Code(err) == codes.NotFound {
					log.Printf("Network-server id %d does not exist !?", config.NetworkServer.Id)
					resp, err := asNetworkServerService.Create(ctx, &asAPI.CreateNetworkServerRequest{
						NetworkServer: &config.NetworkServer,
					})
					if err != nil {
						return fmt.Errorf("grpc: can not create network-server: %v", err)
					}
					config.NetworkServer.Id = resp.Id
					log.Printf("Network-server has been recreated. ID: %v", config.NetworkServer.Id)
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
		config.ServiceProfile.NetworkServerId = config.NetworkServer.Id
		config.ServiceProfile.OrganizationId = config.Organization.Id

		resp, err := asServiceProfileService.List(ctx, &asAPI.ListServiceProfileRequest{
			Limit:          1000,
			OrganizationId: config.Organization.Id,
		})
		if err != nil {
			return fmt.Errorf("grpc: can not list service-profile: %v", err)
		}
		for _, sp := range resp.Result {
			if sp.NetworkServerId == config.NetworkServer.Id {
				if sp.Id != config.ServiceProfile.Id {
					log.Printf("A  service-profile with the same configuration exists? !?. ID: %v <> %v", config.ServiceProfile.Id, sp.Id)
					config.ServiceProfile.Id = sp.Id
					dirty = true
				}
				break
			}
		}

		if config.ServiceProfile.Id == "" {
			resp, err := asServiceProfileService.Create(ctx, &asAPI.CreateServiceProfileRequest{
				ServiceProfile: &config.ServiceProfile,
			})
			if err != nil {
				return fmt.Errorf("grpc: can not create service-profile: %v", err)
			}
			config.ServiceProfile.Id = resp.Id
			log.Printf("Service-profile has been created. ID: %v", config.ServiceProfile.Id)
			dirty = true
		} else {
			resp, err := asServiceProfileService.Get(ctx, &asAPI.GetServiceProfileRequest{
				Id: config.ServiceProfile.Id,
			})
			if err != nil {
				if status.Code(err) == codes.NotFound {
					log.Printf("Service-profile id '%v' does not exist !?", config.ServiceProfile.Id)

					resp, err := asServiceProfileService.Create(ctx, &asAPI.CreateServiceProfileRequest{
						ServiceProfile: &config.ServiceProfile,
					})
					if err != nil {
						return fmt.Errorf("grpc: can not create service-profile: %v", err)
					}
					config.ServiceProfile.Id = resp.Id
					log.Printf("Service-profile has been recreated. ID: %v", config.ServiceProfile.Id)
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
		config.Gateway.OrganizationId = config.Organization.Id
		resp, err := asGatewayService.Get(ctx, &asAPI.GetGatewayRequest{
			Id: config.Gateway.Id,
		})
		if err != nil {
			// log.Printf("\n\n\t-->\tGholi: %q\n\n", err)
			if status.Code(err) == codes.NotFound {
				config.Gateway.NetworkServerId = config.NetworkServer.Id
				config.Gateway.OrganizationId = config.Organization.Id
				_, err = asGatewayService.Create(ctx, &asAPI.CreateGatewayRequest{
					Gateway: &config.Gateway,
				})
				if err != nil {
					return fmt.Errorf("grpc: can not create gateway: %v", err)
				}
				log.Printf("Gateway has been created. ID: %v", config.Gateway.Id)
			} else {
				return fmt.Errorf("grpc: can not get gateway: %v", err)
			}
		} else {
			log.Printf("Gateway %q OK.", resp.Gateway.Name)
		}
	}
	{
		asApplicationService := asAPI.NewApplicationServiceClient(chirpstack)
		config.Application.OrganizationId = config.Organization.Id
		config.Application.ServiceProfileId = config.ServiceProfile.Id
		if config.Application.Id == 0 {
			resp, err := asApplicationService.Create(ctx, &asAPI.CreateApplicationRequest{
				Application: &config.Application,
			})
			if err != nil {
				return fmt.Errorf("grpc: can not create application: %v", err)
			}
			config.Application.Id = resp.Id
			log.Printf("Application has been created. ID: %v", config.Application.Id)
			dirty = true
		} else {
			resp, err := asApplicationService.Get(ctx, &asAPI.GetApplicationRequest{
				Id: config.Application.Id,
			})
			if err != nil {
				if status.Code(err) == codes.NotFound {
					log.Printf("Application id %v does not exist !?", config.Application.Id)
					config.Application.ServiceProfileId = config.ServiceProfile.Id
					resp, err := asApplicationService.Create(ctx, &asAPI.CreateApplicationRequest{
						Application: &config.Application,
					})
					if err != nil {
						return fmt.Errorf("grpc: can not create application: %v", err)
					}
					config.Application.Id = resp.Id
					log.Printf("Application has been recreated. ID: %v", config.Application.Id)
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
		for i, deviceProfile := range config.DeviceProfiles {
			deviceProfile.OrganizationId = config.Organization.Id
			deviceProfile.NetworkServerId = config.NetworkServer.Id
			if deviceProfile.Id == "" {
				resp, err := asDeviceProfileService.Create(ctx, &asAPI.CreateDeviceProfileRequest{
					DeviceProfile: &deviceProfile,
				})
				if err != nil {
					return fmt.Errorf("Err grpc: can not create device-profile: %v", err)
				}
				deviceProfile.Id = resp.Id
				config.DeviceProfiles[i] = deviceProfile
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
						config.DeviceProfiles[i] = deviceProfile
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
	deviceProfileId := config.DeviceProfiles[0].Id
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
				ApplicationId:   config.Application.Id,
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
			ApplicationId:   config.Application.Id,
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
