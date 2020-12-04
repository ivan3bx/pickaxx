
document.addEventListener("DOMContentLoaded", function () {
    var item = document.getElementsByClassName("messages")[0];
    item.scrollTop = item.scrollHeight;
});

function startServer() {
    var xhr = new XMLHttpRequest();
    xhr.open('POST', '/start', true);
    xhr.onload = function () {
        console.log(this.responseText);
    };
    xhr.onreadystatechange = function () {
        if (this.readyState == 4 && this.status > 399) {
            console.log(this.responseText);
        }
    }
    xhr.send("");
};

function stopServer() {
    var xhr = new XMLHttpRequest();
    xhr.open('POST', '/stop', true);
    xhr.onload = function () {
        console.log(this.responseText);
    };
    xhr.onreadystatechange = function () {
        if (this.readyState == 4 && this.status > 399) {
            console.log(this.responseText);
        }
    }
    xhr.send("");
};

function logError(errText) {
    var item = document.getElementsByClassName("message-list")[0];
    item.innerHTML = item.innerHTML + "<li>Error: " + (errText || 'Network request failed') + "</li>";
    item.scrollTop = item.scrollHeight;
}

/*
    websocket handling
*/
if (window["WebSocket"]) {
    conn = new ReconnectingWebSocket("ws://" + document.location.host + "/ws");
    conn.onclose = function (event) {
        // Are we running or not? Unclear.
        document.getElementById("startButton").disabled = false;
        document.getElementById("stopButton").disabled = false;
    }
    conn.onmessage = function (event) {
        var data = JSON.parse(event.data)

        if (data.status !== undefined) {
            if (data.status == "Starting" || data.status == "Running") {
                document.getElementById("startButton").disabled = true;
                document.getElementById("stopButton").disabled = false;
            } else if (data.status == "Stopping") {
                document.getElementById("startButton").disabled = true;
                document.getElementById("stopButton").disabled = true;
            } else {
                document.getElementById("startButton").disabled = false;
                document.getElementById("stopButton").disabled = true;
            }
        }
        if (data.output !== undefined) {
            var item = document.getElementsByClassName("message-list")[0];
            item.innerHTML = item.innerHTML + "<li>" + data.output + "</li>";

            item = document.getElementsByClassName("messages")[0];
            item.scrollTop = item.scrollHeight;
        }
    }
}
