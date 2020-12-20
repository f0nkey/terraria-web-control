document.addEventListener("DOMContentLoaded", function () {
    script()
});

function script() {
    function setCmdStatus(text) {
        let el = document.getElementById("command-status")
        el.innerText = text
        el.classList.remove("anim")
        void el.offsetWidth
        el.classList.add("anim")
    }

    function submitCommand(cmd, callback) {
        let url = window.location.href + "cmd"
        fetch(url, {method: "POST", body: cmd})
            .then(response => response.json())
            .then((data) => {
                if (data.msg === "error") {
                    setCmdStatus("Encountered error: " + data.error)
                    return
                }
                setCmdStatus("Processed command successfully.")
                if(callback !== null && callback !== undefined) callback("success")
            }).catch((e) => {
            setCmdStatus("Encountered error: " + e)
            if(callback !== null && callback !== undefined) callback("fail")
        });
    }

    function sendCommandLineText() {
        let inputText = document.getElementById("command-line").value;
        if(inputText.trim() === "") return;
        printToConsole(inputText)
        submitCommand(inputText)
        if(inputText === "clear") {
            setTimeout(() => {
                if(inputText === "clear") clearConsole()
            }, 250)
        }

        document.getElementById("command-line").value = "";
        document.getElementById("command-line").focus();
    }

    function printToConsole(text) {
        document.getElementById("console").innerHTML += (text + "</br>");
        document.getElementById("console").scrollTo(0, cons.scrollHeight)
    }

    function clearConsole() {
        document.getElementById("console").innerHTML = null;
    }

    document.getElementById("dusk").addEventListener("click", () => {
        submitCommand("dusk")
    })
    document.getElementById("dawn").addEventListener("click", () => {
        submitCommand("dawn")
    })
    document.getElementById("noon").addEventListener("click", () => {
        submitCommand("noon")
    })
    document.getElementById("midnight").addEventListener("click", () => {
        submitCommand("midnight")
    })
    document.getElementById("hard-reset").addEventListener("click", () => {
        if (confirm("This will reboot the server WITHOUT SAVING. Are you sure?")) {
            setCmdStatus("Issuing hard reset ...")
            submitCommand("hardReset", (status) => {
                if(status === "success") {
                    setCmdStatus("Successful. Refreshing page in 3 seconds.")
                    setTimeout(() =>{
                        window.location.reload()
                    }, 3000)
                } else {
                    setCmdStatus("Failed to send hard reset command.")
                }
            })
        }
    })
    document.getElementById("save-world").addEventListener("click", () => {
        submitCommand("save")
    })

    document.getElementById("console-button").addEventListener("click", () => {
        document.getElementById("console-modal").style.display = "block"
        document.getElementById("command-line").focus();
        document.getElementById("console").scrollTo(0, cons.scrollHeight)
    })

    document.getElementById("close").addEventListener("click", () => {
        let modal = document.getElementById("console-modal")
        modal.style.display = "none"
    })

    document.getElementById("submit").addEventListener("click", () => {
        sendCommandLineText()
    })

    document.getElementById("clear-console").addEventListener("click", () => {
        clearConsole()
    })

    document.getElementById("command-line").addEventListener("keyup", function (event) {
        if (event.keyCode === 13) { // enter == 13
            event.preventDefault()
            sendCommandLineText()
        }
    })

    let sock = new WebSocket("ws://" + window.location.href.substr(7) + "console");
    sock.onmessage = (ev) => {
        printToConsole(ev.data)
    }

    sock.onclose = (ev) => {
        printToConsole("Server console communication socket closed.")
    }

    console.log("Loaded")
}
