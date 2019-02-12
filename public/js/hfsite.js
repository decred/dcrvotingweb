// tooltip
var tooltip            = document.getElementById('tooltip');
var tooltipDot         = document.getElementById('tooltip-dot');
var tooltipOptionName  = document.getElementById('tooltip-option-name');
var tooltipOptionValue = document.getElementById('tooltip-option-value');

// agenda progressbar percent bar
var optionProgressBars = document.getElementsByClassName('option-progress');

// indicator icon
var indicatorIcons = document.getElementsByClassName('indicator-icon');

var hideToolTip = function() {
	tooltip.style.display = 'none';
}

var showProgressBarToolTip = function(event) {
	var that = event.target;

	that.addEventListener('mousemove', function (e) {
		tooltip.style.left = e.pageX + "px";
		tooltip.style.top = e.pageY + 25 + "px";
	});

	tooltipOptionName.textContent = that.getAttribute('data-tooltip-text');

	tooltipDot.style.backgroundColor = that.style.backgroundColor;
	tooltipDot.style.display = 'block';

	tooltipOptionValue.textContent = that.getAttribute('data-tooltip-value') + '%';
	tooltipOptionValue.style.display = 'block';

	tooltipOptionName.classList.remove('margin-right-5');

	tooltip.style.display = 'block';
}

var showIndicatorIconToolTip = function(event) {
	var that = event.target;

	that.addEventListener('mousemove', function(e) {
		tooltip.style.left = e.pageX + "px";
		tooltip.style.top = e.pageY + 25 + "px";
	});

	tooltipOptionName.textContent = that.getAttribute('data-tooltip-text');

	tooltipDot.style.display = 'none';
	tooltipOptionValue.style.display = 'none';

	tooltipOptionName.classList.add('margin-right-5');

	tooltip.style.display = 'block';
}

document.addEventListener('DOMContentLoaded', function () {
	for (var i = 0; i < indicatorIcons.length; i++) {
		indicatorIcons[i].addEventListener("mouseover", showIndicatorIconToolTip);
		indicatorIcons[i].addEventListener("mouseout", hideToolTip);
	}

	for (var i = 0; i < optionProgressBars.length; i++) {
		optionProgressBars[i].addEventListener("mouseover", showProgressBarToolTip);
		optionProgressBars[i].addEventListener("mouseout", hideToolTip);
	}
});
