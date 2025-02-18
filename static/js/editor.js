//https://codepen.io/tutsplus/pen/MWZwrGJ
const carousel = document.querySelector(".sliders");
const slide = document.querySelector(".slider-group");

function handleCarouselMove(positive = true) {
    const slideWidth = slide.clientWidth;
    carousel.scrollLeft = positive ? carousel.scrollLeft + slideWidth : carousel.scrollLeft - slideWidth;
}

function showPopup(message, type = "success", duration = 3000) {
    let popup = document.getElementById("popup-notification");

    if (!popup) {
        popup = document.createElement("div");
        popup.id = "popup-notification";
        document.body.appendChild(popup);
    }

    popup.textContent = message;
    popup.className = type;
    popup.classList.add("popup")
    popup.style.display = "block";
    popup.style.opacity = "1";

    setTimeout(() => {
        popup.style.opacity = "0";
        setTimeout(() => { popup.style.display = "none"; }, 1000);
    }, duration);
}
function getFilterValues() {
    return {
        contrast: parseInt($('#contrast').val()),
        vibrance: parseInt($('#vibrance').val()),
        sepia: parseInt($('#sepia').val()),
        vignette: parseInt($('#vignette').val()),
        brightness: parseInt($('#brightness').val()),
        saturation: parseInt($('#saturation').val()),
        exposure: parseInt($('#exposure').val()),
        noise: parseInt($('#noise').val()),
        sharpen: parseInt($('#sharpen').val()),

    };
}

function getFromCloud() {
    $.post('/protected/getimages')
        .done(function (data) {
            if (data.message === "No images found") {
                displayImages(null, false);

            } else {
                console.log('Load success:', data.data);
                localStorage.setItem("cloudImages", JSON.stringify(data.data));
                displayImages(data.data, true);
            }
        })
        .fail(function (jqXHR) {
            console.error('Loading error:', jqXHR.responseText);
            showPopup('Failed to get images.', 'success');
        });

}

function displayImages(images, imagesFound) {
    const container = document.getElementById('images-container');
    container.innerHTML = '';
    if (imagesFound) {
        images.forEach(img => {
            const imgWrapper = document.createElement('div');
            imgWrapper.classList.add('img-wrapper');
            const imgElement = document.createElement('img');
            imgElement.src = "/" + img.filepath;
            imgElement.alt = img.filename;
            imgElement.title = `Created: ${img.created_at}`;
            imgElement.id = "imgElement";

            const editButton = document.createElement('button');
            editButton.textContent = 'Edit';
            editButton.id = 'edit-button';
            editButton.setAttribute('data-image-id', img.id);
            editButton.setAttribute('data-image-filename', img.filename);
            editButton.classList.add("edit", "btn", "btn-success");

            const deleteButton = document.createElement('button');
            deleteButton.textContent = 'Delete';
            deleteButton.id = 'delete-button';
            deleteButton.setAttribute('data-image-id', img.id);
            deleteButton.classList.add("delete", "btn", "btn-warning");



            imgWrapper.appendChild(imgElement);
            imgWrapper.appendChild(editButton);
            imgWrapper.appendChild(deleteButton);

            container.appendChild(imgWrapper);
        });
    } else {
        const message = document.createElement('p');
        message.textContent = "No images found";
        container.appendChild(message);
    }
}


$(function () {
    getFromCloud();

    var canvas = document.getElementById('canvas');
    var ctx = canvas.getContext('2d');

    var img = null;
    var imageLoaded = false;
    var loadedFromCloud = false;
    //Resizing
    function resizeImage(img, maxWidth = 1920, maxHeight = 1080) {
        const ratio = Math.min(maxWidth / img.width, maxHeight / img.height);
        canvas.width = img.width * ratio;
        canvas.height = img.height * ratio;
        ctx.drawImage(img, 0, 0, canvas.width, canvas.height);
    }

    function loadImage(file) {
        revertFilters();

        ctx.clearRect(0, 0, canvas.width, canvas.height);
        canvas.width = 0;
        canvas.height = 0;
        $("#canvas").removeAttr("data-caman-id");

        img = new Image();
        img.crossOrigin = "anonymous";

        if (typeof file === "string") {
            img.onload = function () {
                drawImageOnCanvas(img);
            };
            img.src = file;
            originalFile = img;
            loadedFromCloud = true;
            console.log("Loading from URL:", file);
        } else if (file instanceof File) {
            var reader = new FileReader();
            reader.onload = function (event) {
                img.onload = function () {
                    drawImageOnCanvas(img);
                };
                img.src = event.target.result;
                originalFile = img;
                console.log("Loading from File:", file.name);
            };
            loadedFromCloud = false;
            reader.readAsDataURL(file);
        } else {
            console.error("Invalid file input");
        }
    }

    function drawImageOnCanvas(img) {
        // const resizedCanvas = 
        resizeImage(img);
        // canvas.width = resizedCanvas.width;
        // canvas.height = resizedCanvas.height;

        // ctx.drawImage(resizedCanvas, 0, 0);

        Caman("#canvas", img, function () {
            this.reloadCanvasData();
            this.render();
        });

        imageLoaded = true;

        $("#uploadhide").css("display", "none");
        $("#uploadshow").css("display", "initial");
    }

    $('#uploadbtn').on('change', function (e) {
        var file = e.target.files[0];
        if (file && file.type.startsWith('image/')) {
            loadImage(file);
        } else {
            console.log("Failed to load image");
        }
    });
    $('#uploadnewbtn').on('change', function (e) {
        var file = e.target.files[0];
        if (file && file.type.startsWith('image/')) {
            loadImage(file);
        } else {
            console.log("Failed to load image");
        }
    });


    var cloudImgId = null;
    $(document).on('click', '#edit-button', function (e) {
        const imgId = parseInt(this.getAttribute('data-image-id'));
        const imgFilename = this.getAttribute('data-image-filename');
        const storedData = JSON.parse(localStorage.getItem("cloudImages"));
        var filters = storedData.find((img) => img.id === imgId);

        console.log(imgId);
        console.log(filters.vibrance);
        cloudImgId = imgId;
        setTimeout(() => {
            setSliders(filters.contrast, filters.vibrance, filters.sepia, filters.vignette, filters.brightness, filters.saturation, filters.exposure, filters.noise, filters.sharpen);
        }, 100)
        loadImage(`/images/${imgFilename}`);

    });

    $(document).on('click', '#delete-button', function (e) {
        const imgId = this.getAttribute('data-image-id')
        console.log(imgId)
        $.ajax({
            url: `/protected/delete/${imgId}`,
            type: 'DELETE',
            success: function (data) {
                console.log('DELETE success:', data);
                showPopup('DELETED successfully', "success");
                getFromCloud();
            },
            error: function (jqXHR) {
                console.error('DELETE error:', jqXHR.status);
                showPopup('DELETE FAILED', 'error');
            }
        });

    });




    $('#resetbtn').on('click', function (e) {
        revertFilters();
    });




    $('#savebtn').on('click', function () {
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
    });

    $('#savetocloudbtn').on('click', function () {
        if (imageLoaded) {
            const filters = getFilterValues()
            if (loadedFromCloud) {
                if (cloudImgId) {
                    console.log("image update");

                    $.post('/protected/update', {
                        imageId: cloudImgId,
                        filters: JSON.stringify(filters)
                    })
                        .done(response => {
                            console.log("Update successful:", response);
                            showPopup("Filters updated successfully");
                            getFromCloud();
                        })
                        .fail(error => {
                            console.error("Error updating image:", error);
                            showPopup("There was an error updating the image", 'error');
                        });

                } else {
                    showPopup("There was an error", 'error');
                }
            } else {
                revertFilters();
                setTimeout(() => {
                    canvas.toBlob(function (blob) {
                        const formData = new FormData();
                        formData.append('image', blob, 'original-image.png');
                        formData.append('filters', JSON.stringify(filters));
                        console.log(formData);
                        $.ajax({
                            url: '/protected/upload',
                            type: 'POST',
                            data: formData,
                            processData: false,
                            contentType: false,
                            success: function (data) {
                                if (data.message === "User has exceeded the maximum allowed images") {
                                    showPopup("You have exceeded the maximum allowed images. Please delete an image before uploading a new one.", 'error');
                                } else {
                                    console.log('Upload success:', data);
                                    showPopup('Image and filters uploaded successfully!', 'success');
                                    getFromCloud();
                                    loadedFromCloud = true;
                                    cloudImgId = data.data.imgId;
                                }
                                setSliders(filters.contrast, filters.vibrance, filters.sepia, filters.vignette, filters.brightness, filters.saturation, filters.exposure, filters.noise, filters.sharpen);
                                applyFilters();

                            },
                            error: function (jqXHR) {
                                console.error('Upload error:', jqXHR.status);
                                showPopup('Failed to upload image and filters.', 'error');
                            }
                        });

                    }, 'image/png');
                }, 100);
            }
        }
    });


    function revertFilters() {
        console.log("Reverting filters...");

        $('input[type=range]').val(0);

        if (img && imageLoaded) {
            Caman('#canvas', img, function () {
                this.revert(false);
                this.render();
            });
        } else {
            console.log("No image loaded to revert.");
        }
    }
    let filterTimeout;
    $('input[type=range]').change(() => {
        clearTimeout(filterTimeout);
        filterTimeout = setTimeout(applyFilters, 100);
    });

    function setSliders(contrast, vibrance, sepia, vignette, brightness, saturation, exposure, noise, sharpen) {
        $('#contrast').val(contrast);
        $('#vibrance').val(vibrance);
        $('#sepia').val(sepia);
        $('#vignette').val(vignette);
        $('#brightness').val(brightness);
        $('#saturation').val(saturation)
        $('#exposure').val(exposure)
        $('#noise').val(noise)
        $('#sharpen').val(sharpen)
        applyFilters();
    }
    function applyFilters() {
        if (!imageLoaded) {
            showPopup("Please upload an image first!", 'error');
            revertFilters();
            return;
        }


        const filters = getFilterValues()
        Caman('#canvas', img, function () {
            this.revert(false);
            this.contrast(filters.contrast)
                .vibrance(filters.vibrance)
                .sepia(filters.sepia)
                .vignette(filters.vignette + '%')
                .brightness(filters.brightness)
                .saturation(filters.saturation)
                .exposure(filters.exposure)
                .noise(filters.noise)
                .sharpen(filters.sharpen)
                .render();
        });
    }

});
