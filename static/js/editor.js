
$(function () {
    var canvas = document.getElementById('canvas');
    var ctx = canvas.getContext('2d');

    var img = null;
    var imageLoaded = false;

    function loadImage(file) {
        revertFilters()
        var reader = new FileReader();


        ctx.clearRect(0, 0, canvas.width, canvas.height);
        reader.onload = function (event) {
            img = new Image();
            img.crossOrigin = '';
            img.onload = function () {
                canvas.width = img.width;
                canvas.height = img.height;
                ctx.drawImage(img, 0, 0, img.width, img.height);
                imageLoaded = true;

                $("#uploadhide").css("display", "none");
                $("#uploadshow").css("display", "initial")
            };
            img.src = event.target.result;
        };
        reader.readAsDataURL(file);
    }
    $('#uploadbtn').on('change', function (e) {
        var file = e.target.files[0];
        if (file && file.type.startsWith('image/')) {
            loadImage(file);
        }
    });
    $('#uploadnewbtn').on('change', function (e) {
        var file = e.target.files[0];
        if (file && file.type.startsWith('image/')) {
            revertFilters()
            loadImage(file)
        } else {
            console.log('failed to load')
        }
    })
    var $reset = $('#resetbtn');
    var $noise = $('#noisebtn');
    var $savetocloud = $('#savetocloudbtn')

    var $hdr = $('#hdrbtn');
    var $save = $('#savebtn');

    $('input[type=range]').change(applyFilters);
    function revertFilters() {
        $('input[type=range').val(0)
    }
    function applyFilters() {
        if (!imageLoaded) {
            alert("Please upload an image first!");
            revertFilters();
            return;
        }
        var hue = parseInt($('#hue').val());
        var cntrst = parseInt($('#contrast').val());
        var vibr = parseInt($('#vibrance').val());
        var sep = parseInt($('#sepia').val());
        var vig = parseInt($('#vignette').val());
        var bri = parseInt($('#brightness').val());

        Caman('#canvas', img, function () {
            this.revert(false);
            this.hue(hue).contrast(cntrst).vibrance(vibr).sepia(sep).vignette(vig + '%').brightness(bri).render();
        });
    }

    $reset.on('click', function (e) {
        revertFilters()
        Caman('#canvas', img, function () {
            this.revert(false);
            this.render();
        });
    });

    $noise.on('click', function (e) {
        Caman('#canvas', img, function () {
            this.noise(10).render();
        });
    });

    $hdr.on('click', function (e) {
        Caman('#canvas', img, function () {
            this.contrast(10);
            this.contrast(10);
            this.jarques();
            this.render();
        });
    });

    $save.on('click', function () {
        if (imageLoaded) {
            Caman('#canvas', function () {
                this.render(function () {
                    const dataURL = canvas.toDataURL('image/png');

                    const link = document.createElement('a');
                    link.href = dataURL;
                    link.download = 'edited-image.png';

                    link.click();
                });
            });
        }
    })
    $savetocloud.on('click', function () {
        if (imageLoaded) {
            const filters = {
                hue: parseInt($('#hue').val()),
                contrast: parseInt($('#contrast').val()),
                vibrance: parseInt($('#vibrance').val()),
                sepia: parseInt($('#sepia').val()),
                vignette: parseInt($('#vignette').val()),
                brightness: parseInt($('#brightness').val())
            };

            canvas.toBlob(function (blob) {
                const formData = new FormData();
                formData.append('image', blob, 'original-image.png');
                formData.append('filters', JSON.stringify(filters));

                fetch('/protected/upload', {
                    method: 'POST',
                    body: formData,
                })
                    .then(response => response.json())
                    .then(data => {
                        console.log('Upload success:', data);
                        alert('Image and filters uploaded successfully!');
                    })
                    .catch(error => {
                        console.error('Upload error:', error);
                        alert('Failed to upload image and filters.');
                    });
            }, 'image/png');
        }
    });



});
