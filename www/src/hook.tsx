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
    Tooltip,
    FormGroup,
    Input,
    InputAdornment,
    CardContent,
    CardHeader,
    Card
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
        const lorawanMeta = {
            profile: "WaziDev",
        };
        setDevice((device: Device) => ({
            ...device,
            meta: {
                ...device.meta,
                lorawan: lorawanMeta
            }
        }));
        wazigate.setDeviceMeta(device.id, {
            lorawan: lorawanMeta
        })
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
        textAlign: "center",
    },
    scrollBox: {
        padding: theme.spacing(2),
        minWidth: "fit-content",
        maxWidth: 650,
        display: "inline-block",
        textAlign: "left",
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
        width: 300,
    },
    longInput: {
        width: 500,
    },
    footer: {
        color: "#34425a",
    },
    cardBtn: {
        marginRight: theme.spacing(2),
    }
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

    const [newMeta, setNewMeta] = useState<LoRaWANMeta>(null);

    const setMeta = (meta: LoRaWANMeta) => {
        setDevice((device: Device) => ({
            ...device,
            meta: {
                ...device.meta,
                lorawan: meta
            }
        }));
    }

    const hasUnsavedChanges = newMeta != null;

    const handleProfileChange = (event: React.ChangeEvent<HTMLSelectElement>) => {
        setNewMeta({
            ...newMeta,
            profile: event.target.value as string
        });
    };
    const handleDevAddrChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        const devAddr = event.target.value as string;
        setNewMeta({
            ...newMeta,
            devEUI: devAddr2EUI(devAddr),
            devAddr: devAddr
        });
    };
    const handleNwkSKeyChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        setNewMeta({
            ...newMeta,
            nwkSEncKey: event.target.value as string
        });
    };
    const handleAppKeyChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        setNewMeta({
            ...newMeta,
            appSKey: event.target.value as string
        });
    };

    const handleDevEUIChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        setNewMeta({
            ...newMeta,
            devEUI: event.target.value as string
        });
    };

    const handleRemoveClick = () => {
        if (confirm("Do you want to remove the LoRaWAN settings from this device?")) {
            wazigate.setDeviceMeta(device.id, {
                lorawan: null
            }).then(() => {
                setMeta(null);
                setNewMeta(null);
            }, (error) => {
                // TODO: improve
                alert(error);
            });
        }
    }

    const submitChanges = () => {
        wazigate.setDeviceMeta(device.id, {
            lorawan: {
                ...meta,
                ...newMeta
            }
        }).then(() => {
            setMeta(newMeta);
            setNewMeta(null);
        }, (error) => {
            // TODO: improve
            alert(error);
        });
    }

    const resetChanges = () => {
        setNewMeta(null);
    }

    const generateKeys = () => {
        const r = () => "0123456789ABCDEF"[Math.random() * 16 | 0];
        const rk = () => {
            var k = new Array(32);
            for (var i = 0; i < 32; i++) k[i] = r();
            return k.join("");
        }
        setNewMeta({
            ...newMeta,
            nwkSEncKey: rk(),
            appSKey: rk()
        });
    }

    const devAddr2EUI = (devAddr: string) => "AA555A00" + devAddr;

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

    if (!meta) {
        return null;
    }

    // TODO: UI inputs should show an error if the keys or devAddr is bad formatted
    return (
        <div className={classes.root}><div className={classes.scrollBox}>
            <Card className={classes.paper}>
                <CardHeader avatar={<RouterIcon />}
                    action={
                        <IconButton aria-label="remove" onClick={handleRemoveClick}>
                            <RemoveIcon />
                        </IconButton>
                    }
                    title="LoRaWAN Settings"
                />
                <CardContent>
                    <FormGroup>
                        <FormControl className={classes.shortInput}>
                            <InputLabel id="lorawan-profile-label">LoRaWAN Profile</InputLabel>
                            <Select
                                labelId="lorawan-profile-label"
                                id="lorawan-profile"
                                value={newMeta?.profile || meta?.profile || ""}
                                onChange={handleProfileChange}
                            >
                                <MenuItem value="WaziDev">WaziDev</MenuItem>
                                <MenuItem value="Other">Other</MenuItem>
                            </Select>
                        </FormControl><br />
                        {(newMeta?.profile || meta?.profile) === "WaziDev" ? (
                            <Fragment>
                                <FormControl>
                                    <InputLabel htmlFor="lorawan-devaddr">DevAddr (Device Address)</InputLabel>
                                    <Input
                                        id="lorawan-devaddr"
                                        onChange={handleDevAddrChange}
                                        value={newMeta?.devAddr || meta?.devAddr || ""}
                                        className={classes.shortInput}
                                        endAdornment={
                                            <InputAdornment position="end">
                                                <Tooltip title="Autogenerate">
                                                    <IconButton size="small" onClick={generateDevAddr}>
                                                        <AddIcon />
                                                    </IconButton>
                                                </Tooltip>
                                            </InputAdornment>
                                        }
                                    />
                                </FormControl>
                                <FormControl>
                                    <InputLabel htmlFor="lorawan-nwskey">NwkSKey (Network Session Key)</InputLabel>
                                    <Input
                                        id="lorawan-nwskey"
                                        onChange={handleNwkSKeyChange}
                                        value={newMeta?.nwkSEncKey || meta?.nwkSEncKey || ""}
                                        className={classes.longInput}
                                        endAdornment={
                                            <InputAdornment position="end">
                                                <Tooltip title="Autogenerate">
                                                    <IconButton size="small" onClick={generateKeys}>
                                                        <AddIcon />
                                                    </IconButton>
                                                </Tooltip>
                                            </InputAdornment>
                                        }
                                    />
                                </FormControl>
                                <TextField
                                    id="lorawan-appkey"
                                    label="AppKey (App Key)"
                                    onChange={handleAppKeyChange}
                                    value={newMeta?.appSKey || meta?.appSKey || ""}
                                    className={classes.longInput} />
                            </Fragment>
                        ) : (
                            <TextField
                                id="lorawan-deveui"
                                label="Device EUI"
                                onChange={handleDevEUIChange}
                                value={newMeta?.devEUI || meta?.devEUI || ""}
                                className={classes.longInput} />
                        )}
                    </FormGroup>
                </CardContent>
                <CardActions className={classes.footer}>
                    <Grow in={hasUnsavedChanges}>
                        <Button
                            className={classes.cardBtn}
                            variant="contained"
                            color="primary"
                            onClick={submitChanges}
                            startIcon={<SaveIcon />}
                        >
                            Save
                            </Button>
                    </Grow>
                    <Grow in={hasUnsavedChanges} timeout={({ enter: 500, exit: 200 })}>
                        <Button
                            className={classes.cardBtn}
                            onClick={resetChanges}
                        >
                            Reset
                            </Button>
                    </Grow>
                </CardActions>
            </Card>
        </div></div>
    );
});

// Hook scripts always need to call this function to signal that the hook file was
// successfully executed.
hooks.resolve();