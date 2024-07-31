function showLoading() {
    var loader = document.getElementById("loader");
    loader.style.display = "block"; 
}

function hideLoading() {
    var loader = document.getElementById("loader");
    loader.style.display = "none";
}

function updateCountryOptions() {
    var areaSelect = document.getElementById("areaSelect");
    var selectedValue = areaSelect.value;
    var countrySelect = document.getElementById("countrySelect");

    fetch("/web/getCountryOptions?areaOption=" + selectedValue)
        .then(response => response.json())
        .then(data => {
            countrySelect.innerHTML = "";
            data.forEach(function(option) {
                var opt = document.createElement("option");
                opt.value = option.Value;
                opt.text = option.Text;
                countrySelect.add(opt);
            });
            countrySelect.dispatchEvent(new Event('change'));
        });
}

function updateNodeOptions() {
    var countrySelect = document.getElementById("countrySelect");
    var selectedValue = countrySelect.value;
    var nodeSelect = document.getElementById("nodeSelect");

    fetch("/web/getNodeOptions?countryOption=" + selectedValue)
        .then(response => response.json())
        .then(data => {
            nodeSelect.innerHTML = "";
            data.forEach(function(option) {
                var opt = document.createElement("option");
                opt.value = option.Value;
                opt.text = option.Text;
                nodeSelect.add(opt);
            });
        });
}

function submitForm(event) {
    event.preventDefault(); // 阻止默认表单提交行为
    var nodeSelect = document.getElementById("nodeSelect");
    var nodeID = nodeSelect.value;
    if (nodeID.trim() === '') {
        alert("Please select a node");
        return;
    }
    
    showLoading()

    var url = "/change?id="+nodeID;
    fetch(url, {method: 'POST'})
         .then(response => {
            return response.text().then(text => {
                return { status: response.status, body: text };
            });
        })
        .then(data => {
            hideLoading()
            if (data.status === 200) {
                location.reload();
            } else {
                alert(data.body)
            }
        })
        .catch(error => {
            hideLoading()
            alert('Form submission failed.');
        });
    
}

// var node = "{{.Node}}";
document.addEventListener('DOMContentLoaded', function() {
    // var data = "{{ .Node }}";
    var currentNodeID = nodeID;
    var currentNodeAreaID = nodeAreaID;
    var values = currentNodeAreaID.split("-");
    var area = values[0]
    var country = area + "-" + values[1];

    var areaSelect = document.getElementById("areaSelect");
    for (let i = 0; i < areaSelect.options.length; i++) {
        let option = areaSelect.options[i];
        if (option.value === area) {
            areaSelect.selectedIndex = i;
            break
        }
    }

    var countrySelect = document.getElementById("countrySelect");
    fetch("/web/getCountryOptions?areaOption=" + area)
        .then(response => response.json())
        .then(data => {
            countrySelect.innerHTML = "";
            data.forEach(function(option) {
                var opt = document.createElement("option");
                opt.value = option.Value;
                opt.text = option.Text;
                countrySelect.add(opt);

                if (option.Value === country) {
                    countrySelect.selectedIndex = countrySelect.options.length -1;
                }
            });
         });

    var nodeSelect = document.getElementById("nodeSelect");
    fetch("/web/getNodeOptions?countryOption=" + country)
        .then(response => response.json())
        .then(data => {
            nodeSelect.innerHTML = "";
            data.forEach(function(option) {
                var opt = document.createElement("option");
                opt.value = option.Value;
                opt.text = option.Text;
                nodeSelect.add(opt);

                if (option.Value === currentNodeID) {
                    nodeSelect.selectedIndex = nodeSelect.options.length -1
                }
            });
        });
});