function init() {
    document.getElementById("status").style = "display: none"

}

const delay = 500;
let   i     = 0;
function load(validate_fn) {
    if (validate_fn() != true) {
        return;
    }

    if ((doc = document.getElementById("form")) != null) {
        doc.style = "display: none"
    }
    if (document.getElementById("message").innertext != "") {
        document.getElementById("status").style = "display: inline"
    }
    document.getElementById("loading").innerText = "Loading" + ('.'.repeat(i % 4))
    i++;
    setTimeout(load, delay, validate_fn)
}
