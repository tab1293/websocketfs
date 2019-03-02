import tus from 'tus-js-client'

var file;
var ws;

var wsFileInput = document.getElementById('wsFileInput');
wsFileInput.onchange = function(e) {
    var startTime, endTime;

    if (!ws) {
        return false;
    }

    file = e.target.files[0];
    var fileAnnounce = {
        type: 'fileAnnounce',
        size: file.size,
        lastModified: file.lastModified,
        mime: file.type,
        name: file.name
    };

    ws.send(JSON.stringify(fileAnnounce));

}

var tusFileInput = document.getElementById('tusFileInput');
tusFileInput.onchange = function(e) {
    var startTime, endTime;
    //Get the selected file from the input element
    var file = e.target.files[0]

    // Create a new tus upload
    var upload = new tus.Upload(file, {
        endpoint: "http://35.186.181.47:8018/files/",
        retryDelays: [0, 3000, 5000, 10000, 20000],
        metadata: {
            filename: file.name,
            filetype: file.type
        },
        onError: function(error) {
            console.log("Failed because: " + error)
        },
        onProgress: function(bytesUploaded, bytesTotal) {
            var percentage = (bytesUploaded / bytesTotal * 100).toFixed(2)
            console.log(bytesUploaded, bytesTotal, percentage + "%")
        },
        onSuccess: function() {
            endTime = new Date();
            var timeDiff = endTime - startTime; //in ms
            // strip the ms
            timeDiff /= 1000;

            // get seconds
            var seconds = Math.round(timeDiff);
            console.log(seconds + " seconds");
            console.log("Download %s from %s", upload.file.name, upload.url)
        }
    })

    // Start the upload
    startTime = new Date();
    upload.start()
}

ws = new WebSocket(`ws://35.186.181.47:8015/ws`);
// ws = new WebSocket(`ws://localhost:8015/ws`);
ws.onmessage = function(e) {
    console.log('onmessage start')
    var message = JSON.parse(e.data);

    if (!file) {
        return;
    }

    var readRequest = message;
    var reader = new FileReader();
    reader.onloadend = function() {
        var data = reader.result.substring('data:application/octet-stream;base64,'.length);
        var readResponse = {
            type: 'readResponse',
            file_name: readRequest.file_name,
            data: data,
            offset: readRequest.offset,
        };
        console.log(`sending data ${data.length}`);
        ws.send(JSON.stringify(readResponse));
    }
    console.log(`load requested for bytes ${readRequest.offset} - ${readRequest.offset + readRequest.length} with length ${readRequest.length}`)
    var blob = file.slice(readRequest.offset, readRequest.offset + readRequest.length);
    reader.readAsDataURL(blob);
    console.log('exit onmessage')
}

ws.onopen = function(e) {
    console.log('ws open');
}

ws.onclose = function(e) {
    console.log('ws close')
}