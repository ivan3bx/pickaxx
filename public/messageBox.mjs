const websocketURL = "ws://" + document.location.host + "/ws";

export { init, resetScroll, clear };

var messages = null;
var messageList = null;
var conn = null;
var startBtn = null;
var stopBtn = null;

function init(startButton, stopButton) {
    startBtn = startButton;
    stopBtn = stopButton;

    messages = document.querySelector(".messages");
    messageList = document.querySelector(".message-list");

    conn = new ReconnectingWebSocket(websocketURL, null, {
        debug: false,
        reconnectInterval: 400
    });

    conn.onmessage = handleMessage;
    conn.onclose = handleClose;

}

// Handle Websocket messages.
// Two types of responses are returned:
//
// 1. Server output:
//      { "output" : "text that will appear in the messages-list" }
//
// 2. Process status changes:
//      { "status" : "Starting | Stopping | etc.." }
//
function handleMessage(event) {
    let data = JSON.parse(event.data)

    if (data.status !== undefined) {
        if (data.status == "Starting" || data.status == "Running") {
            startBtn.disabled = true;
            stopBtn.disabled = false;
        } else if (data.status == "Stopping") {
            startBtn.disabled = true;
            stopBtn.disabled = true;
        } else {
            startBtn.disabled = false;
            stopBtn.disabled = true;
        }
    } else if (data.output !== undefined) {
        let li = document.createElement("li");

        li.appendChild(document.createTextNode(data.output));
        messageList.appendChild(li);

        resetScroll();
    } else {
        console.log("event undefined:" + JSON.stringify(event));
    }
}

function handleClose() {
    startBtn.disabled = false;
    stopBtn.disabled = false;
};

function clear() {
    removeAllChildNodes(messageList)
}

function resetScroll() {
    messages.scrollTop = messages.scrollHeight;
}

function removeAllChildNodes(parent) {
    while (parent.firstChild) {
        parent.removeChild(parent.firstChild);
    }
}
