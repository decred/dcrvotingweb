$(document).ready(function() {



	var time = 100,
	    viewport = $(window),
	    active = 'active',
	    inactive = 'inactive',

	    //agenda
	    agenda = $('.agenda'),

	    // agenda progressbar percent bar
	    optionProgress = $('.option-progress'),
	    totalVotesCast = $('.total-votes-cast'),

	    // tooltip
	    tooltip = $('.tooltip'),
	    	tooltipDot = $('.tooltip-dot').hide(),
	    	tooltipSquare = $('.tooltip-square').hide(),
	    	tooltipOptionName = $('.tooltip-option-name'),
	    	tooltipOptionValue = $('.tooltip-option-value'),

	    // indicator icon
	    indicatorIcon = $('.indicator-icon'),

	    // charts desktop toggle button
	    powPosChartToggle = $('.chart-toggle-side');

	// charts draw
	function drawTheChart(ChartData, ChartOptions, chartId, ChartType) {
		var myChart = new Chart(document.getElementById(chartId).getContext('2d'),
		    {
		    	type: ChartType,
		    	data: ChartData,
		    	options: ChartOptions
		    }
		 );
		document.getElementById(chartId).getContext('2d').stroke();
	}



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