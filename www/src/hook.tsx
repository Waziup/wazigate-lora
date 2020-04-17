import React, { Fragment, useState } from "react";
import RemoveIcon from '@material-ui/icons/Remove';
import RouterIcon from '@material-ui/icons/Router';
import SaveIcon from '@material-ui/icons/Save';
import AddIcon from '@material-ui/icons/AddCircleOutline';
import { Device, Waziup, Sensor, Actuator, DeviceHook, DeviceHookProps, DeviceMenuHook, MenuHookProps, HookRegistry } from "waziup";
import {
    Grow,
    IconButton,
    ListItemIcon,
    ListItemText,
    Toolbar,
    Typography,
    makeStyles,
    InputLabel,
    FormControl,
    TextField,
    MenuItem,
    Select,
    Paper,
    Button,
    CardActions,
    Tooltip
} from '@material-ui/core';

// Add a item to the dashboard menu.
// The menu structure will be like this:
// ```
// - Dashboard
// - Settings
// + Apps
//   - LoRaWAN    <- new
// ```
hooks.setMenuHook("apps.lorawan", {
    primary: "LoRaWAN",
    icon: <RouterIcon />,
    href: "#/apps/waziup.wazigate-lora/index.html",
});

// A DeviceMenuHook adds a item to the devices context menu.
// We show a "Make LoRaWAN" for all devices that don't have `lorawan` metadata.  
hooks.addDeviceMenuHook((props: DeviceHookProps & MenuHookProps) => {
    const {
        device,
        handleMenuClose,
        setDevice
    } = props;
    const handleClick = () => {
        handleMenuClose();
        setDevice((device: Device) => ({
            ...device,
            meta: {
                ...device.meta,
                lorawan: {
                    profile: "",
                }
            }
        }));
        // gateway.setDeviceMeta(device.id, {
        //     lorawan: {
        //         DevEUI: null,
        //     }
        // })
    }

    if (device === null || device.meta.lorawan) {
        return null;
    }

    return (
        <MenuItem onClick={handleClick} key="waziup.wazigate-lora">
            <ListItemIcon>
                <RouterIcon fontSize="small" />
            </ListItemIcon>
            <ListItemText primary="Make LoRaWAN" secondary="Declare as LoRaWAN device" />
        </MenuItem>
    );
})

const useStylesLoRaWAN = makeStyles((theme) => ({
    root: {
        overflow: "auto",
    },
    scrollBox: {
        padding: theme.spacing(2),
        minWidth: "fit-content",
    },
    paper: {
        background: "#d8dee9",
        minWidth: "fit-content",
    },
    header: {
        color: "#34425a",
    },
    name: {
        flexGrow: 1,
    },
    body: {
        padding: theme.spacing(2),
    },
    shortInput: {
        width: "200px",
    },
    longInput: {
        width: 400,
        maxWidth: "100%",
    },
    button: {
        margin: "16px 8px 0px",
    },
    footer: {
        color: "#34425a",
    },
}));

type LoRaWANMeta = {
    profile: string;
    devEUI: string;
    devAddr: string;
    appSKey: string;
    nwkSEncKey: string;
};

// A DevicHook adds some UI to a device.
// We add some input fields to make LoRaWAN settings for a device with `lorawan` meta.
hooks.addDeviceHook((props: DeviceHookProps) => {
    const classes = useStylesLoRaWAN();
    const {
        device,
        setDevice
    } = props;

    const meta = device?.meta["lorawan"] as LoRaWANMeta;

    const setMeta = (meta: LoRaWANMeta) => {
        setDevice((device: Device) => ({
            ...device,
            meta: {
                ...device.meta,
                lorawan: meta
            }
        }));
    }

    const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false);

    const handleProfileChange = (event: React.ChangeEvent<{ value: unknown }>) => {
        setMeta({
            ...meta,
            profile: event.target.value as string
        });
        setHasUnsavedChanges(true);
    };
    const handleDevAddrChange = (event: React.ChangeEvent<{ value: unknown }>) => {
        const devAddr = event.target.value as string;
        setMeta({
            ...meta,
            devEUI: devAddr2EUI(devAddr),
            devAddr: devAddr
        });
        setHasUnsavedChanges(true);
    };
    const handleNwkSKeyChange = (event: React.ChangeEvent<{ value: unknown }>) => {
        setMeta({
            ...meta,
            nwkSEncKey: event.target.value as string
        });
        setHasUnsavedChanges(true);
    };
    const handleAppKeyChange = (event: React.ChangeEvent<{ value: unknown }>) => {
        setMeta({
            ...meta,
            appSKey: event.target.value as string
        });
        setHasUnsavedChanges(true);
    };

    const handleRemoveClick = () => {
        if(confirm("Do you want to remove the LoRaWAN settings from this device?")) {
            wazigate.setDeviceMeta(device.id, {
                lorawan: null
            }).then(() => {
                setMeta(null);
            }, (error) => {
                // TODO: improve
                alert(error);
            });
        }
    }

    const saveChanges = () => {
        wazigate.setDeviceMeta(device.id, {
            lorawan: meta
        }).then(() => {
            setHasUnsavedChanges(false);
        }, (error) => {
            // TODO: improve
            alert(error);
        });
    }

    const generateKeys = () => {
        const r = () => "0123456789ABCDEF"[Math.random()*16|0];
        const rk = () => {
            var k = new Array(32);
            for(var i=0;i<32;i++) k[i] = r();
            return k.join("");
        }
        setMeta({
            ...meta,
            nwkSEncKey: rk(),
            appSKey: rk()
        });
        setHasUnsavedChanges(true);
    }

    const devAddr2EUI = (devAddr: string) => "AA555A00"+devAddr;

    const generateDevAddr = async () => {
        // TODO: implement :)
        // The randomDevAddr endpoint exists but requires a valid devEUI
        // but the devEUI if generated from the devAddr, so this conflicts
        alert("This feature is not available right now.");
        // const devEUI = "AA555A0012345678";
        // const devAddr = await wazigate.set<string>("apps/waziup.wazigate-lora/randomDevAddr", devEUI);
        // setMeta({
        //     ...meta,
        //     devAddr: devAddr
        // });
    }

    // TODO: UI inputs should show an error if the keys or devAddr is bad formatted

    return (
        <div className={classes.root}><div className={classes.scrollBox}>
        <Grow in={!!meta} key="waziup.wazigate-lora">
        <Paper variant="outlined" className={classes.paper}>
            <Toolbar className={classes.header} variant="dense">
                <IconButton edge="start">
                    <RouterIcon />
                </IconButton>
                <Typography variant="h6" noWrap className={classes.name}>
                    LoRaWAN Settings
                </Typography>
                <IconButton onClick={handleRemoveClick}>
                    <RemoveIcon />
                </IconButton>
            </Toolbar>
            <div className={classes.body}>
                <FormControl className={classes.shortInput}>
                    <InputLabel id="lorawan-profile-label">LoRaWAN Profile</InputLabel>
                    <Select
                        labelId="lorawan-profile-label"
                        id="lorawan-profile"
                        value={meta?.profile||""}
                        onChange={handleProfileChange}
                    >
                        <MenuItem value="WaziDev">WaziDev</MenuItem>
                        <MenuItem value="">Other</MenuItem>
                    </Select>
                </FormControl><br />
                { meta?.profile === "WaziDev" ? (
                    <Fragment>
                        <TextField
                            id="lorawan-devaddr"
                            label="DevAddr (Device Address)"
                            onChange={handleDevAddrChange}
                            value={meta?.devAddr||""}
                            className={classes.shortInput} />
                        <Tooltip title="Autogenerate">
                            <IconButton size="small" className={classes.button} onClick={generateDevAddr}>
                                <AddIcon />
                            </IconButton>
                        </Tooltip>
                        <br />
                        <TextField
                            id="lorawan-nwskey"
                            label="NwkSKey (Network Session Key)"
                            onChange={handleNwkSKeyChange}
                            value={meta?.nwkSEncKey||""}
                            className={classes.longInput}/>
                        <Tooltip title="Autogenerate">
                            <IconButton size="small" className={classes.button} onClick={generateKeys}>
                                <AddIcon />
                            </IconButton>
                        </Tooltip>
                        <br />
                        <TextField
                            id="lorawan-appkey"
                            label="AppKey (App Key)"
                            onChange={handleAppKeyChange}
                            value={meta?.appSKey||""}
                            className={classes.longInput}/>
                    </Fragment>
                ): null }
            </div>
            <Grow in={hasUnsavedChanges}>
                <CardActions className={classes.footer}>
                    <Button
                        startIcon={<SaveIcon />}
                        onClick={saveChanges}
                    >
                        Save
                    </Button>
                </CardActions>
            </Grow>
        </Paper>
        </Grow>
        </div></div>
    );
});

// Hook scripts always need to call this function to signal that the hook file was
// successfully executed.
hooks.resolve();