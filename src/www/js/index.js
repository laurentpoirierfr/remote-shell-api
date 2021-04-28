


$("#execute_id").on("click", function (event) {
    var id = $("#product_id").val();
    var shell = $("#shell_id").val();
    var parameters = $("#parameters_id").val();

    url = "/api/v1/commands/" + id + "/" + shell + "?" + parameters;
    console.log(url);

    $.get(url, function (data) {

        console.log(data);

        $("#console_id").html(data);

        var maxLine = 24;	
		var terminal = [];

        e = new EventSource(data.channel);
        e.onmessage = function (event) {
            console.log(event.data);

            if (terminal.length == maxLine - 1 ) {
				terminal.shift();
			}
			terminal.push(event.data + '<br>');
			var bash = "";
			terminal.forEach(item => {
				bash += item;
			})

            document.getElementById("console_id").innerHTML = bash;
        };

    });


});



