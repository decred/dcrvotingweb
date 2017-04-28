$(document).ready(function() {

	// tooltip
	optionProgress.add(totalVotesCast).add(indicatorIcon).mouseover(function(e) {
		var that = $(this);

		that.on('mousemove', function(e) {
		      tooltip.css({
		         	left: e.pageX,
		      	top: e.pageY + 25
			});
		});

		tooltipOptionName.text(that.attr('data-tooltip-text'));
		tooltipOptionValue.text(that.attr('data-tooltip-value')+'%').show();
		tooltipDot.css('background-color', that.css('background-color')).show();

		if(that.is(optionProgress)) {
			tooltipSquare.hide();
			tooltipOptionName.removeClass('margin-right-5');
		}
		if(that.is(totalVotesCast)) {
			tooltipDot.hide();
			tooltipSquare.show();
			tooltipOptionName.removeClass('margin-right-5');
		}
		if(that.is(indicatorIcon)) {
			tooltipDot.add(tooltipSquare).add(tooltipOptionValue).hide();
			tooltipOptionName.addClass('margin-right-5');
		}

		tooltip.show();
	}).mouseout(function() {
		tooltip.hide();
	});

});