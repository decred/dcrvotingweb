( function ( $ ){
  $.fn.accordionjs = function(options) {
    //Version check
    if (typeof jQuery != 'undefined') {
      var version = jQuery.fn.jquery;
      var major   = parseInt(version.substr(0, version.indexOf('.')));
      if (3 > major){
        throw "Error requires a minimum of jQuery 3.0";
      }
    }

    var _self = this;
    var _selfId = "#"+_self[0].id;
    var defaults = {
      steps: [],
      keyNavigation: true
    };
    var _settings = $.extend(defaults, options);

    /*
    ** BEGIN: Private functions.
    */

    function pad(n, w) { return Array(Math.max(w - String(n).length +1, 0)).join(0)+n; }

    function _getStepById(userStepId) {
      var _internalStep = userStepId - 1;
      if (_internalStep < 0 || _internalStep > _settings.steps.lengh)
      {
        throw "Step out of range";
      }
      return _internalStep;
    }

    function _installKeyHandler()
    {
      if (false === _settings.keyNavigation){
        return;
      }

      $(_selfId+"> li").on("keydown", function (e) { 
        e = e || window.event;
        var keyCode = e.keyCode || e.which;
        var arrow = {left: 37, up: 38, right: 39, down: 40 };
        if (e.ctrlKey && e.altKey) {
          switch (keyCode) {
            case arrow.left:
              _gotoPreviousTab();
              break;
            case arrow.right:
              _gotoNextTab();
              break;
          } // ENd SWITCH
        }// END IF ctrl & alt
      });
    }

    function _gotoNextTab(){
      console.log("_gotoNextTab");
      var _current = _self.select();
      console.log("current " +_current);
      if (_current < (_settings.steps.length+1)) {
        ++_current;
        _self.select(_current);
      }
    }

    function _gotoPreviousTab(){
      console.log("_gotoPreviousTab");
      var _current = _self.select();
      if (_current > 1){
        --_current;
        _self.select(_current);
      }
    }


    function setSizes(){
      var tabWidth = $(".accordionjs-select:first").outerWidth();
      var areaWidth = $(_selfId).width();
      var sliderWidth = areaWidth - ((_settings.steps.length) * tabWidth) - (_settings.steps.length+1) ;
      var contentWidth = sliderWidth;

      $(".accordionjs > li").css("margin-right", "-"+sliderWidth+"px");      
      $(".accordionjs-content").css("width", contentWidth+"px");

      $(_selfId+"-styles").remove();
      $("head").append("<style id='"+_selfId+"-styles'>.accordionjs-select:checked ~ .accordionjs-separator { margin-right: "+sliderWidth+"px;}</style>");

      /* Heights */
      var tabHeight = $(_selfId).outerHeight();
      $(".accordionjs-title").css("height", tabHeight+"px");
      $(".accordionjs-select").css("height", tabHeight+"px");
      $(".accordionjs-content").css("height", tabHeight+"px");
    }

    function _createHTMLForStep(node, step){
      var selected = (step.selected ? 'checked="checked"' : "");
      var icon     = "fa-square-o";
      var disabled = "";

      switch(step.status) {
        case "complete":
          icon = "fa-check-square-o";
          break;
        case "disabled":
          icon = "fa-lock";
          disabled = "disabled";
          break;
        case "optional":
          icon = "fa-minus-square-o";
          break;          
      }
      
      var oldContents = $(node).contents();
      $(node).empty();
      $(node).attr("tabindex", step.number);
      $(node).append('<input type="radio" name="ac-tab" value="'+step.number+'" class="accordionjs-select" '+selected+' ' + disabled + '/>');
      $(node).append('<div class="accordionjs-title">' +
                    '  <span><i class="fa '+icon+' fa-rotate-90"></i><i class="jobStep fa fa-rotate-90 ">'+pad(step.number+1, 2)+'</i>'+step.title+'</span>' +
                    '</div>');
      $(node).append('<div class="accordionjs-content" id="accordionjs-page-'+step.number+'"></div>');
      $(node).append('<div class="accordionjs-separator"></div>');

      $("#accordionjs-page-"+step.number).append(oldContents);

    }

    function _init() {
      $(_selfId).addClass("accordionjs");
      $(_selfId+" li").each(function(index, node){
        var step = {
          number: index,
          title: "Step "+index,
          required: false,
          completed: false,
          status: "incomplete",
          selected: false
        }
        
        step.title    = ($(node).data("title")    ?  $(node).data("title")    : "Step " + (step.number+1));
        step.required = ($(node).data("required") ?  $(node).data("required") : false);
        step.selected = ($(node).data("selected") ?  $(node).data("selected") : false);
        step.status   = ($(node).data("status")   ?  $(node).data("status")   : "incomplete");
        if (0 == step.number) {
          step.selected = true;
        }

        _settings.steps[index] = step;
        _createHTMLForStep(node, step)
      });
      
      _installKeyHandler();
    }

    /*
    ** BEGIN: Public functions.
    */

    this.select = function(stepId){
      var _selectedStepId = 0;      

      //If we haven't requested a step to select, return the current.
      if (typeof stepId === 'undefined'){
        _selectedStepId = $(".accordionjs-select:checked").val();
      }
      else
      {
        _selectedStepId = _getStepById(stepId);
        $('input[name="ac-tab"][value="'+parseInt(_selectedStepId)+'"]').prop("checked", "checked");
      }

      return ++_selectedStepId; //Normalise for 0 indexed arrays.
    }


    /*!
    ** Sets a step as completed by providing the step number
    **/   
    this.complete = function(stepId){
      var _internalStep = _getStepById(stepId);

      $(".accordionjs-title:eq("+(_internalStep)+") i:first").removeClass("fa-square-o")
                                                             .removeClass("fa-minus-square-o")
                                                             .addClass("fa-check-square-o");
    }

    this.enable = function(stepId){
      var _internalStep = _getStepById(stepId);
      $(".accordionjs-title:eq("+(_internalStep)+") i:first").removeClass("fa-lock")
                                                             .addClass("fa-square-o");
      $(".accordionjs-select:eq("+(_internalStep)+")").removeAttr("disabled");
    }

    this.disable = function(stepId){
      var _internalStep = _getStepById(stepId);

      $(".accordionjs-title:eq("+(_internalStep)+") i:first").removeClass("fa-square-o")
                                                             .removeClass("fa-minus-square-o")
                                                             .addClass("fa-lock");

      $(".accordionjs-select:eq("+(_internalStep)+")").attr("disabled", "disabled");
    }

    /*
    ** BEGIN: Initialise
    */
    _init();
    setSizes();

    /*
    ** @todo Find a more efficient mechanism to do this. 
    */
    $(window).resize(function(e){
      setSizes();
    });

    return this;
  };
}( jQuery) );

