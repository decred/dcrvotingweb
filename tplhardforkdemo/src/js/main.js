$('#showboxpow').click(function() {
    $('#boxpow').toggleClass('hidden');
    $('#boxpos').addClass('hidden');
    $('#boxstake').addClass('hidden');
//    $('.boxindicators').addClass('hidden');
});
$('#showboxpos').click(function() {
    $('#boxpos').removeClass('hidden');
    $('#boxpow').addClass('hidden');
    $('#boxstake').addClass('hidden');
//    $('.boxindicators').addClass('hidden');
});
$('#showboxstake').click(function() {
    $('#boxstake').toggleClass('hidden');
    $('#boxpos').addClass('hidden');
    $('#boxpow').addClass('hidden');
//    $('.boxindicators').addClass('hidden');
});

$('.radial-progress').click(function() {
    $('#boxpow').toggleClass('hidden');
    $('#boxpos').addClass('hidden');
    $('#boxstake').addClass('hidden');
//    $('.boxindicators').addClass('hidden');
});

$('.radial-progress2').click(function() {
     $('#boxpos').toggleClass('hidden');
    $('#boxpow').addClass('hidden');
    $('#boxstake').addClass('hidden');
//    $('.boxindicators').addClass('hidden');
});

$('.radial-progress3').click(function() {
    $('#boxstake').toggleClass('hidden');
    $('#boxpos').addClass('hidden');
    $('#boxpow').addClass('hidden');
//    $('.boxindicators').addClass('hidden');
});