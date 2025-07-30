package app

import (
	"context"
	"fmt"
	"log"

	asAPI "github.com/chirpstack/chirpstack/api/go/v4/api"
	"github.com/chirpstack/chirpstack/api/go/v4/common"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type APIToken string

var chirpstackUsername = "admin"        // default
var chirpstackPassword = "admin"        // default
var chirpstackTenantName = "ChirpStack" // Use the default "ChirpStack" tenant for WaziGate

func connectToChirpStack() (*grpc.ClientConn, error) {
	var err error

	conn, err := grpc.Dial("waziup.wazigate-lora.chirpstack-v4:8080",
		grpc.WithBlock(),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("grpc: can not dial: %v", err)
	}

	internalClient := asAPI.NewInternalServiceClient(conn)
	loginReq := &asAPI.LoginRequest{
		Email:    chirpstackUsername,
		Password: chirpstackPassword,
	}
	res, err := internalClient.Login(context.Background(), loginReq)
	if err != nil {
		return nil, fmt.Errorf("grpc: can not login: %v", err)
	}
	token := res.Jwt
	if err := conn.Close(); err != nil {
		return nil, fmt.Errorf("grpc: can not close connection: %v", err)
	}

	conn, err = grpc.Dial("waziup.wazigate-lora.chirpstack-v4:8080",
		grpc.WithBlock(),
		grpc.WithPerRPCCredentials(APIToken(APIToken(token))),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("grpc: can not dial: %v", err)
	}

	return conn, nil
}

// func refreshChirpstackToken() {
// 	for {
// 		time.Sleep(chirpstackTokenRefreshInterval)
// 		err := connectToChirpStack()
// 		if err != nil {
// 			log.Printf("grpc: connect failed: %v", err)
// 			continue
// 		}
// 	}
// }

func InitChirpstack() error {

	conn, err := connectToChirpStack()
	if err != nil {
		return fmt.Errorf("grpc: can not connect to ChirpStack: %v", err)
	}
	defer conn.Close()

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
		{
			asTenantServiceClient := asAPI.NewTenantServiceClient(conn)

			req := &asAPI.ListTenantsRequest{
				Limit:  100,
				Offset: 0,
				Search: "",
				UserId: "",
			}
			// Fetch list of tenants
			resp, err := asTenantServiceClient.List(ctx, req)
			if err != nil {
				return fmt.Errorf("grpc: can not list tenants: %v", err)
			}

			/*
				// Print the response
				log.Printf("ListTenantsResponse:")
				log.Printf("TotalCount: %d\n", resp.TotalCount)
				log.Println("Tenants:")
				for _, tenant := range resp.Result {
					log.Printf("ID: %s, Name: %s\n", tenant.Id, tenant.Name)
				}
			*/

			// Iterate through the tenants list to find the tenant with the name "ChirpStack"
			var tenantId string
			// Get the ID of chirpstack_tenantName
			for _, tenant := range resp.Result {
				if tenant.Name == chirpstackTenantName {
					tenantId = tenant.Id
					break
				}
			}

			if tenantId == "" {
				log.Printf("Cannot find tenant id for %s", chirpstackTenantName)
			}

			// Set Config.Tenant.Id to the id of the "ChirpStack" tenant
			Config.Tenant.Id = tenantId

			dirty = true

			//log.Printf("Config.Tenant.Id set to: %d", Config.Tenant.Id)
			log.Printf("Tenant %q OK.", chirpstackTenantName)
		}
	}
	{
		{
			asGatewayService := asAPI.NewGatewayServiceClient(conn)
			Config.Gateway.TenantId = Config.Tenant.Id

			if Config.Gateway.GatewayId == "" {
				_, err := asGatewayService.Create(ctx, &asAPI.CreateGatewayRequest{
					Gateway: &Config.Gateway,
				})
				if err != nil {
					return fmt.Errorf("grpc: can not create gateway: %v", err)
				}
				log.Printf("Gateway has been created. ID: %v", Config.Gateway.GatewayId)
				dirty = true
			} else {
				resp, err := asGatewayService.Get(ctx, &asAPI.GetGatewayRequest{
					GatewayId: Config.Gateway.GatewayId,
				})
				if err != nil {
					// log.Printf("\n\n\t-->\tGholi: %q\n\n", err)
					if status.Code(err) == codes.NotFound {
						_, err = asGatewayService.Create(ctx, &asAPI.CreateGatewayRequest{
							Gateway: &Config.Gateway,
						})
						if err != nil {
							return fmt.Errorf("grpc: can not create gateway: %v", err)
						}
						log.Printf("Gateway has been created. ID: %v", Config.Gateway.GatewayId)
					} else {
						return fmt.Errorf("grpc: can not get gateway: %v", err)
					}
				} else {
					log.Printf("Gateway %q OK.", resp.Gateway.Name)
				}
			}
		}
	}
	{
		asApplicationService := asAPI.NewApplicationServiceClient(conn)
		Config.Application.TenantId = Config.Tenant.Id

		resp, err := asApplicationService.List(ctx, &asAPI.ListApplicationsRequest{
			Limit:    1000,
			TenantId: Config.Tenant.Id,
		})
		if err != nil {
			return fmt.Errorf("grpc: can not list application-profile: %v", err)
		}
		for _, a := range resp.Result {
			if a.Id != Config.Application.Id {
				log.Printf("A application with the same configuration exists? !?. ID: %v <> %v", Config.Application.Id, a.Id)
				Config.Application.Id = a.Id
				Config.Application.Name = a.Name
				dirty = true
			}
			break
		}

		if Config.Application.Id == "" {
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
		asDeviceProfileService := asAPI.NewDeviceProfileServiceClient(conn)
		for i, deviceProfile := range Config.DeviceProfiles {
			if deviceProfile.Id == "" {
				deviceProfile := asAPI.DeviceProfile{
					Name:                "Wazidev",
					TenantId:            Config.Tenant.Id,
					MacVersion:          common.MacVersion_LORAWAN_1_0_1,
					RegParamsRevision:   common.RegParamsRevision_A,
					Region:              common.Region_EU868,
					PayloadCodecRuntime: asAPI.CodecRuntime_CAYENNE_LPP,
					PayloadCodecScript:  "CAYENNE_LPP",
				}

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
						deviceProfile := asAPI.DeviceProfile{
							Name:                "Wazidev",
							TenantId:            Config.Tenant.Id,
							MacVersion:          common.MacVersion_LORAWAN_1_0_1,
							RegParamsRevision:   common.RegParamsRevision_A,
							Region:              common.Region_EU868,
							PayloadCodecRuntime: asAPI.CodecRuntime_CAYENNE_LPP,
							PayloadCodecScript:  "CAYENNE_LPP",
						}
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

	conn, err := connectToChirpStack()
	if err != nil {
		return fmt.Errorf("grpc: can not connect to ChirpStack: %v", err)
	}
	defer conn.Close()

	deviceProfileId := Config.DeviceProfiles[0].Id
	deviceClient := asAPI.NewDeviceServiceClient(conn)
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
				SkipFcntCheck:   true,
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
			SkipFcntCheck:   true,
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

	conn, err := connectToChirpStack()
	if err != nil {
		return fmt.Errorf("grpc: can not connect to ChirpStack: %v", err)
	}
	defer conn.Close()

	deviceClient := asAPI.NewDeviceServiceClient(conn)
	r, err := deviceClient.GetActivation(ctx, &asAPI.GetDeviceActivationRequest{
		DevEui: devEUI,
	})
	//log.Printf("Err: %v", err)
	if err == nil {
		//log.Printf("Error is nill!!!!!")
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

// //////////////////////////////////////////////////////////////////////////////

func (a APIToken) GetRequestMetadata(ctx context.Context, url ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": fmt.Sprintf("Bearer %s", a),
	}, nil
}

func (a APIToken) RequireTransportSecurity() bool {
	return false
}
