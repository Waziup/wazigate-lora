import './main.css';

if (navigator.platform.indexOf("Win") == 0) {
    document.body.classList.add("windows");
} else if (navigator.platform.indexOf("Mac") == 0) {
    document.body.classList.add("mac");
} else if (navigator.platform.indexOf("Linux") != -1) {
    document.body.classList.add("linux");
}

var btnSubmit = document.getElementById("submit");
var selectRegion = <HTMLSelectElement> document.getElementById("region");
var selectModulation = <HTMLSelectElement> document.getElementById("modulation");
var selectFrequency = <HTMLSelectElement> document.getElementById("frequency");
var selectSpreading = <HTMLSelectElement> document.getElementById("spreading");
var selectBandwidth = <HTMLSelectElement> document.getElementById("bandwidth");
var selectCodingRate = <HTMLSelectElement> document.getElementById("codingrate");

btnSubmit.addEventListener("click", () => {
    var value = {
        region: selectRegion.value,
        modulation: selectModulation.value,
        frequency: parseInt(selectFrequency.value),
        spreading: parseInt(selectSpreading.value),
        bandwidth: parseInt(selectBandwidth.value),
        codingrate: parseInt(selectCodingRate.value)
    };
    console.log(value);
})