$('.radial-progress').click(function() {
$('#boxpow').toggleClass("hidden");
$('#boxpos').addClass( "hidden" );
$('#boxstake').addClass( "hidden" );
});
$('.radial-progress2').click(function() {
$('#boxpos').toggleClass("hidden");
$('#boxpow').addClass( "hidden" );
$('#boxstake').addClass( "hidden" );
});
$('.radial-progress3').click(function() {
$('#boxstake').toggleClass("hidden");
$('#boxpos').addClass( "hidden" );
$('#boxpow').addClass( "hidden" );
});
$(document).keydown(function(event){
    var display = "0";
    var keycode = (event.keyCode ? event.keyCode : event.which);
    if(keycode === 49){
        $('#boxpow').toggleClass("hidden");
        $('#boxpos').addClass( "hidden" );
        $('#boxstake').addClass( "hidden" );
    }
    if(keycode === 50){
        $('#boxpos').toggleClass("hidden");
        $('#boxpow').addClass( "hidden" );
        $('#boxstake').addClass( "hidden" );
        console.log('key 2 pressed')
    }
    if(keycode === 51){
        $('#boxstake').toggleClass("hidden");
        $('#boxpos').addClass( "hidden" );
        $('#boxpow').addClass( "hidden" );
    }
    if(keycode === 37){
        $('#boxstake').removeClass("hidden");
        $('#boxpos').removeClass( "hidden" );
        $('#boxpow').removeClass( "hidden" );
    }
    if(keycode === 39){
        $('#boxstake').addClass("hidden");
        $('#boxpos').addClass( "hidden" );
        $('#boxpow').addClass( "hidden" );
    }
});