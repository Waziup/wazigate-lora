import React, { Fragment, useState } from "react";
import RemoveIcon from '@material-ui/icons/Remove';
import RouterIcon from '@material-ui/icons/Router';
import BluetoothIcon from '@material-ui/icons/Bluetooth';
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
                    DevEUI: null,
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
        background: "lightblue",
        minWidth: "fit-content",
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
}));

// A DevicHook adds some UI to a device.
// We add some input fields to make LoRaWAN settings for a device with `lorawan` meta.
hooks.addDeviceHook((props: DeviceHookProps) => {
    const classes = useStylesLoRaWAN();
    const {
        device,
        setDevice
    } = props;

    const [profile, setProfile] = useState("");

    const lorawan = device?.meta["lorawan"];

    const handleProfileChange = (event: React.ChangeEvent<{ value: unknown }>) => {
        setProfile(event.target.value as string);
    };

    const handleRemoveClick = () => {
        if(confirm("Do you want to remove the LoRaWAN settings from this device?")) {
            setDevice((device: Device) => ({
                ...device,
                meta: {
                    ...device.meta,
                    lorawan: undefined
                }
            }));
        }
    }

    return (
        <div className={classes.root}><div className={classes.scrollBox}>
        <Grow in={!!lorawan} key="waziup.wazigate-lora">
        <Paper variant="outlined" className={classes.paper}>
            <Toolbar variant="dense">
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
                        value={profile}
                        onChange={handleProfileChange}
                    >
                        <MenuItem value="WaziDev">WaziDev</MenuItem>
                        <MenuItem value="">Other</MenuItem>
                    </Select>
                </FormControl><br />
                { profile === "WaziDev" ? (
                    <Fragment>
                        <TextField id="lorawan-devaddr" label="DevAddr (Device Address)" className={classes.shortInput}/><br />
                        <TextField id="lorawan-nwskey" label="NwkSKey (Network Session Key)" className={classes.longInput}/><br />
                        <TextField id="lorawan-appkey" label="AppKey (App Key)" className={classes.longInput}/>
                    </Fragment>
                ): null }
            </div>
        </Paper>
        </Grow>
        </div></div>
    );
});

// Hook scripts always need to call this function to signal that the hook file was
// successfully executed.
hooks.resolve();