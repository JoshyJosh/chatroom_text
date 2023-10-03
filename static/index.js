console.log("establishing websocket");

WS = new WebSocket('wss://127.0.0.1:8080/websocket/');
WS.onopen = (event) => {
    console.log("onopen event: ", event);
    WS.send('{"msg":"entered chat"}');
};
WS.onerror = (event) => {
    console.log("onerror event: ", event);
};
WS.onmessage = (event) => {
    console.log("onmessage event: ", event);
};
WS.onclose = (event) => {
    console.log("onclose event: ", event);
};
