// If JavaScript is enabled remove 'no-js' class and give 'js' class
jQuery('html').removeClass('no-js').addClass('js');

// Add .osx class to html if on Os/x
if ( navigator.appVersion.indexOf("Mac")!=-1 ) 
	jQuery('html').addClass('osx');

// When DOM is fully loaded
jQuery(document).ready(function($) {

	(function() {

	/* --------------------------------------------------------	
		Twitter bootstrap - carousel, tooltip, popover 
	   --------------------------------------------------------	*/	

		// initialize carousel
		$('[rel=carousel]').carousel();
		// initialize tooltip
		$('[rel=tooltip]').tooltip();
		// initialize popover
		$('[rel=popover]').popover();


	    $('.accordion').on('show', function (e) {
	         $(e.target).prev('.accordion-heading').find('.accordion-toggle').addClass('active');
	    });

	    $('.accordion').on('hide', function (e) {
	        $(this).find('.accordion-toggle').not($(e.target)).removeClass('active');
	    });


	/* --------------------------------------------------------	
		External links
	   --------------------------------------------------------	*/	

	    $(window).load(function() {

			$('a[rel=external]').attr('target','_blank');
			
		});

	})();


/* --------------------------------------------------------	
	Flickr feed
   --------------------------------------------------------	*/	

	(function() {

		$('.flickr-feed').each(function(){

			var flickr_id 	   = $(this).data('flickr-id'),
				flickr_limit   = $(this).data('flickr-limit')	? $(this).data('flickr-limit') 		: 12,
				flickr_tags    = $(this).data('flickr-tags')   	? $(this).data('flickr-tags') 		: '',
				flickr_tagmode = $(this).data('flickr-tagmode')	? $(this).data('flickr-tagmode')	: 'any';

			$(this).jflickrfeed({

				limit: flickr_limit,
				qstrings: {
					id: flickr_id,
					tags: flickr_tags,
					tagmode: flickr_tagmode
				},
				itemTemplate: '<a href="{{link}}" rel="external"><img src="{{image_s}}" alt="{{title}}" /></a>'

			});
		})

	})();

/* --------------------------------------------------------	
	Zoom and link overlays (e.g. for thumbnails)
   --------------------------------------------------------	*/	

	(function() {

		$(window).load(function() {

			$('.link').each(function(){
				var $this = $(this);
				var $height = $this.find('img').height();
				var span = $('<span>').addClass('link-overlay').html('&nbsp;').css('top',$height/2).click(function(){
					if (href = $this.find('a:first').attr('href')) {
						top.location.href=href;
					}
				});
				$this.append(span);
			})
			$('.zoom').each(function(){
				var $this = $(this);
				var $height = $this.find('img').height();
				var span = $('<span>').addClass('zoom-overlay').html('&nbsp;').css('top',$height/2);
				$this.append(span);
			})

		});

	})();

/* --------------------------------------------------------	
	Responsible navigation
   --------------------------------------------------------	*/	
	
	(function() {

		var $mainNav    = $('.navbar .nav'),
			responsibleNav = '<option value="" selected>Navigate...</option>';

		// Responsive nav
		$mainNav.find('li').each(function() {
			var $this   = $(this),
				$link = $this.children('a'),
				depth   = $this.parents('ul').length - 1,
				indent  = '';

			if( depth ) {
				while( depth > 0 ) {
					indent += ' - ';
					depth--;
				}
			}

			if ($link.text())
				responsibleNav += '<option ' + ($this.hasClass('active') ? 'selected="selected"':'') + ' value="' + $link.attr('href') + '">' + indent + ' ' + $link.text() + '</option>';

		}).end().after('<select class="nav-responsive">' + responsibleNav + '</select>');

		$('.nav-responsive').on('change', function() {
			window.location = $(this).val();
		});
			
	})();


/* --------------------------------------------------------	
	Portfolio 
   --------------------------------------------------------	*/	

   (function() {

		$(window).load(function(){

			// container
			var $container = $('#portfolio-items');

			function filter_projects(tag)
			{
			  // filter projects
			  $container.isotope({ filter: tag });
			  // clear active class
			  $('#portfolio-filter li.active').removeClass('active');
			  // add active class to filter selector
			  $('#portfolio-filter').find("[data-filter='" + tag + "']").parent().addClass('active');
			  // update location hash
			  if (tag!='*')
				window.location.hash=tag.replace('.','');
			  if (tag=='*')
			  	window.location.hash='';
			}

			if ($container.length) {

				// conver data-tags to classes
				$('.project').each(function(){
					$this = $(this);
					var tags = $this.data('tags');
					if (tags) {
						var classes = tags.split(',');
						for (var i = classes.length - 1; i >= 0; i--) {
							$this.addClass(classes[i]);
						};
					}
				})

				// initialize isotope
				$container.isotope({
				  // options...
				  itemSelector : '.project',
				  layoutMode   : 'fitRows'
				});

				// filter items
				$('#portfolio-filter li a').click(function(){
					var selector = $(this).attr('data-filter');
					filter_projects(selector);
					return false;
				});

				// filter tags if location.has is available. e.g. http://example.com/work.html#design will filter projects within this category
				if (window.location.hash!='')
				{
					filter_projects( '.' + window.location.hash.replace('#','') );
				}

			}
		})

	})();


/* --------------------------------------------------------
	Back to top button
   --------------------------------------------------------	*/

	(function() {

   			$('<i id="back-to-top"></i>').appendTo($('body'));

			$(window).scroll(function() {

				if( $(this).scrollTop() > $(window).height()/5) {
					$("#back-to-top:not(.shown)").addClass('shown');
				} else {
					$('#back-to-top').removeClass('shown');
				}

			});

			$('#back-to-top').click(function() {
				$('body,html').animate({scrollTop:0},600);
			});

	})();

/* --------------------------------------------------------	
	Contact-form
   --------------------------------------------------------	*/	

   (function() {

   		$('#contact-form-submit').data('original-text', $('#contact-form-submit').html() );

		$('#contact-form-submit').click(function(e){

			var form = $('#contact-form').serialize();

			$('#contact-form-submit').addClass('disabled').html('Sending ...');

			setTimeout(function(){

				// reset message field
				$('#contact-form-response').hide().attr('class','alert');

				// post form data using ajax
				$.post( 'php/contact-form.php', form, 

					function(response) {

						// reset contact form button with original text
						$('#contact-form-submit').removeClass('disabled').html( $('#contact-form-submit').data('original-text') );

						// email was sent
						if ( response.status == 1 ) {

							message = '<i class="icon-ok"></i> <b>Thank You!</b> <br />Thanks for leaving your message. We will get back to you soon.';
							$('#contact-form-response').addClass('alert-success');

						// there were errors, show them
						} else {

							message = '' + response.errors;
							$('#contact-form-response').addClass('alert-warning');

						}

						// show response message
						$('#contact-form-response').show().html(message);

					}
				,"json");

			},300);

		})

	})();

/* --------------------------------------------------------	
	Newsletter form
   --------------------------------------------------------	*/	

   (function() {

   		$('#newsletter-form').submit(function(e){

			var form = $('#newsletter-form').serialize();

			$('#newsletter-form').hide();
			$('.newsletter .ajax-loader').show();

			setTimeout(function(){
			// post form data using ajax
			$.post( 'php/newsletter-form.php', form, 

				function(response) {

					$('.newsletter .ajax-loader').hide();

					// email was sent
					if ( response.status == 1 ) {
						
						$('#newsletter-form').html("&#10004; Thanks, you have been subscribed!");
						$('#newsletter-form').show();

					// there were errors, show them
					} else {

						$('#newsletter-form').show();
						alert(response.errors);
					}
				}
			,"json");

			},600);

			// prevent from reloading a page. 
			e.preventDefault();
		})

	})();

/* --------------------------------------------------------	
	Swipe support for slider
   --------------------------------------------------------	*/	

   (function() {

   		var is_touch_device = !!('ontouchstart' in window);

		function swipe( e, direction ) {

			var $carousel = $(e.currentTarget);
			
			if( direction === 'left' )
				$carousel.find('.carousel-control.right').trigger('click');
			
			if( direction === 'right' )
				$carousel.find('.carousel-control.left').trigger('click');
		}
		
		if (is_touch_device === true) {

			$('#carousel').swipe({
				allowPageScroll : 'auto',
				swipeLeft       : swipe,
				swipeRight      : swipe
			});

		}

	})();

/* --------------------------------------------------------	
	Keyboard shortcuts
   --------------------------------------------------------	*/	

   (function() {

		$('a[rel=shortcut]').each(function(){

			var $this = $(this);
			var key = $this.data('key');
			var href = $this.attr('href');

			if (key && href) {
				$(document).bind('keydown', key, function(){
					top.location.href = href;
				});
			}
		})

	})();


})