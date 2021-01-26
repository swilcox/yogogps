const source = new EventSource('/sse');

source.onmessage = function showData(event) {
    // event.data will always return a string of our event response
    // in this case, our server sends us a stringified JSON object 
    // that we can parse below
    const data = JSON.parse(event.data);
    console.log(data);
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
        gridField.innerText = "EM65ms";
    }
    if (data.class == "SKY") {
        const satelliteData = document.getElementById('satellite-data');
        sHTML = '';
        for (sat of data.satellites) {
            sHTML += '<tr>';
            sHTML += '<td>' + sat.PRN + '</td>';
            sHTML += '<td>' + sat.el + '</td>';
            sHTML += '<td>' + sat.az + '</td>';
            sHTML += '<td>' + sat.ss + '</td>';
            sHTML += '<td>' + sat.used + '</td>';
            sHTML += '</tr>';
        }        
        satelliteData.innerHTML = sHTML;
    }
    //const temperatureOutputEl = document.getElementById('temperature-output');
    //const updatedAtOutputEl = document.getElementById('updated-at');

    //temperatureOutputEl.innerText = data.temperature;
    //updatedAtOutputEl.innerText = data.updatedAt;
}
