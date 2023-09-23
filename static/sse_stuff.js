
const source = new EventSource('/sse');

const GNSSMAP = {
    0: {
        "flag": "US",
        "code": "GP"
    },
    1: {
        "flag": "US",
        "code": "SB"
    },
    2: {
        "flag": "EU",
        "code": "GA"
    },
    3: {
        "flag": "CN",
        "code": "BD"
    },
    4: {
        "flag": "JP",
        "code": "IM"
    },
    5: {
        "flag": "JP",
        "code": "QZ"
    },
    6: {
        "flag": "RU",
        "code": "GL"
    },
    7: {
        "flag": "IN",
        "code": "IR"
    }
}

function getFlagEmoji(countryCode) {
  const codePoints = countryCode
    .toUpperCase()
    .split('')
    .map(char =>  127397 + char.charCodeAt());
  return String.fromCodePoint(...codePoints);
}

source.onmessage = function showData(event) {
    // event.data will always return a string of our event response
    // in this case, our server sends us a stringified JSON object 
    // that we can parse below
    const data = JSON.parse(event.data);
    // console.log(data);
    if (data.class == "TPV") {        
        const timeField = document.getElementById('time-value');
        const latField = document.getElementById('lat-value');
        const lonField = document.getElementById('lon-value');
        const altField = document.getElementById('alt-value');
        const gridField = document.getElementById('grid-square');
        timeField.innerText = data.time;
        latField.innerText = data.lat;
        lonField.innerText = data.lon;
        altField.innerText = data.alt + "m";
        gridField.innerText = data.GridSquare;
    }
    if (data.class == "SKY") {
        const satelliteData = document.getElementById('satellite-data');
        sHTML = '';
        for (sat of data.satellites) {
            //console.log(sat);
            sHTML += '<tr>';
            sHTML += '<td>' + getFlagEmoji(GNSSMAP[sat.gnssid].flag) + ' ' + GNSSMAP[sat.gnssid].code + ' ' + sat.svid + '</td>';
            sHTML += '<td>' + sat.PRN + '</td>';
            sHTML += '<td>' + sat.el + '</td>';
            sHTML += '<td>' + sat.az + '</td>';
            sHTML += '<td>' + sat.ss + '</td>';
            sHTML += '<td>' + sat.used + '</td>';
            sHTML += '<td>' + sat.health + '</td>';
            sHTML += '</tr>';
        }        
        satelliteData.innerHTML = sHTML;
    }
    //const temperatureOutputEl = document.getElementById('temperature-output');
    //const updatedAtOutputEl = document.getElementById('updated-at');

    //temperatureOutputEl.innerText = data.temperature;
    //updatedAtOutputEl.innerText = data.updatedAt;
}
