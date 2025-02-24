//Funkcia pre carousel scrollovanie filtrov https://webdesign.tutsplus.com/easy-slider-carousel-with-pure-css--cms-107626t
const carousel = document.querySelector(".sliders");
const slide = document.querySelector(".slider-group");

function handleCarouselMove(positive = true) {
    const slideWidth = slide.clientWidth;
    carousel.scrollLeft = positive ? carousel.scrollLeft + slideWidth : carousel.scrollLeft - slideWidth;
}
//Funkcia pre zobrazovanie popupu so správou (Príklad pre error)
function showPopup(message, type = "success", duration = 3000) {
    let popup = document.getElementById("popup-notification");

    if (!popup) {
        popup = document.createElement("div");
        popup.id = "popup-notification";
        document.body.appendChild(popup);
    }

    popup.textContent = message;
    popup.className = type;
    popup.classList.add("popup");
    popup.style.display = "block";
    popup.style.opacity = "1";

    setTimeout(() => {
        popup.style.opacity = "0";
        setTimeout(() => { popup.style.display = "none"; }, 1000);
    }, duration);
}
//funckia zobrazí správu zatiaľ čo sa filtre budú renderovať
function showLoading(show) {
    const loadingElement = document.getElementById("loading");
    if (loadingElement) {
        loadingElement.style.display = show ? "flex" : "none";
    }
}
//Funkcia pre získanie hodnôt filtrov zo sliderov
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
//Funkcia pre získanie obrázkov uložených na serveri
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
//Funkcia pre zobrazenie obrázkov získaných z funkcie getFromCloud():45
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
    //Premenné
    const canvas = document.getElementById('canvas');
    const ctx = canvas.getContext('2d');

    let img = null;
    let imageLoaded = false;
    let loadedFromCloud = false;
    let cloudImgId = null;
    //Zmení veľkosť obrázka
    function resizeImage(img, maxWidth = 1920, maxHeight = 1080) {
        const ratio = Math.min(maxWidth / img.width, maxHeight / img.height);
        canvas.width = img.width * ratio;
        canvas.height = img.height * ratio;
        ctx.drawImage(img, 0, 0, canvas.width, canvas.height);
    }
    //Vyčistí canvas a načíta obrázok na canvas
    function loadImage(file, callback) {
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
                if (callback) callback();
            };
            img.src = file;
            loadedFromCloud = true;
            console.log("Loading from URL:", file);
        } else if (file instanceof File) {
            const reader = new FileReader();
            reader.onload = function (event) {
                img.onload = function () {
                    drawImageOnCanvas(img);
                    if (callback) callback();
                };
                img.src = event.target.result;
                console.log("Loading from File:", file.name);
            };
            loadedFromCloud = false;
            reader.readAsDataURL(file);
        } else {
            console.error("Invalid file input");
        }
    }
    //Vykreslí obrázok na canvas
    function drawImageOnCanvas(img) {
        resizeImage(img);
        try {
            Caman("#canvas", img, function () {
                this.reloadCanvasData()
                this.render();
            });
        } catch (error) {
            console.error("Caman.js initialization failed:", error);
            showPopup("Failed to initialize image editor.", "error");
        }

        imageLoaded = true;
        $("#uploadhide").css("display", "none");
        $("#uploadshow").css("display", "initial");
    }
    //Uploadne obrázok do UI
    function handleFileUpload(e) {
        const file = e.target.files[0];
        if (file && file.type.startsWith('image/')) {
            loadImage(file);
        } else {
            console.log("Failed to load image");
        }
    }
    //Navráti základné hodnoty sliderov
    function revertFilters(callback) {
        console.log("Reverting filters...");
        $('input[type=range]').val(0);

        if (img && imageLoaded) {
            Caman('#canvas', img, function () {
                this.revert(false)
                this.render(() => {
                    if (callback) callback()
                });
            });
        } else {
            console.log("No image loaded to revert.");
        }
    }

    //Nastaví hodnoty sliderov na požadované hodnoty
    function setSliders(contrast, vibrance, sepia, vignette, brightness, saturation, exposure, noise, sharpen) {
        $('#contrast').val(contrast);
        $('#vibrance').val(vibrance);
        $('#sepia').val(sepia);
        $('#vignette').val(vignette);
        $('#brightness').val(brightness);
        $('#saturation').val(saturation);
        $('#exposure').val(exposure);
        $('#noise').val(noise);
        $('#sharpen').val(sharpen);
    }

    //Aplikuje filtre na canvas
    function applyFilters() {
        if (!imageLoaded) {
            showPopup("Please upload an image first!", 'error');
            revertFilters();
            return;
        }
        const filters = getFilterValues();
        console.log("applying filters", filters);
        showLoading(true);
        Caman('#canvas', img, function () {
            this.revert(false)
            this.contrast(filters.contrast)
                .vibrance(filters.vibrance)
                .sepia(filters.sepia)
                .brightness(filters.brightness)
                .saturation(filters.saturation)
                .exposure(filters.exposure)
                .noise(filters.noise)
                .sharpen(filters.sharpen)
                .vignette(filters.vignette)
                .render(() => {
                    showLoading(false);
                }
                );
        });
    }
    //Button eventy

    //Aplikuje filtre pri zmene sliderov
    $('input[type=range]').change(applyFilters);
    //Nahranie obrázka do UI pre úpravy
    $('#uploadbtn, #uploadnewbtn').on('change', handleFileUpload);
    //Naloaduje obrázok pre následné úpravy
    $(document).on('click', '#edit-button', function (e) {
        const imgId = parseInt(this.getAttribute('data-image-id'));
        const imgFilename = this.getAttribute('data-image-filename');
        const storedData = JSON.parse(localStorage.getItem("cloudImages"));
        const filters = storedData.find((img) => img.id === imgId);

        cloudImgId = imgId;
        loadImage(`/images/${imgFilename}`, function () {
            setSliders(filters.contrast, filters.vibrance, filters.sepia, filters.vignette, filters.brightness, filters.saturation, filters.exposure, filters.noise, filters.sharpen);
            applyFilters();
        });
    });
    //Pošle DELETE request pre odstránenie obrázka
    $(document).on('click', '#delete-button', function (e) {
        const imgId = this.getAttribute('data-image-id');
        console.log(imgId);
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
    //Resetuje všetky filtre nastavené na obrázku
    $('#resetbtn').on('click', revertFilters);
    //Stiahne upravený obrázok do zariadenia
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
    //Pošle POST request pre uloženie obrázka na server,
    //alebo pri práci na obrázku z cloudu nahradí filtre
    $('#savetocloudbtn').on('click', function () {
        if (imageLoaded) {
            const filters = getFilterValues();
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
                revertFilters(function () {
                    canvas.toBlob(function (blob) {
                        const formData = new FormData();
                        formData.append('image', blob, 'original-image.png');
                        formData.append('filters', JSON.stringify(filters));
                        fetch('/protected/upload', {
                            method: 'POST',
                            body: formData,
                            mode: 'same-origin',

                        })
                            .then(response => {
                                if (!response.ok) {
                                    throw new Error(`HTTP error! status: ${response.status}`);
                                }
                                return response.json();
                            })
                            .then(data => {
                                if (data.message === "User has exceeded the maximum allowed images") {
                                    showPopup("You have exceeded the maximum allowed images. Please delete an image before uploading a new one.", 'error');
                                    setSliders(filters.contrast, filters.vibrance, filters.sepia, filters.vignette, filters.brightness, filters.saturation, filters.exposure, filters.noise, filters.sharpen);
                                    applyFilters();
                                } else {
                                    console.log('Upload success:', data);
                                    showPopup('Image and filters uploaded successfully!', 'success');
                                    getFromCloud();
                                    loadedFromCloud = true;
                                    cloudImgId = data.data.imgId;
                                    setSliders(filters.contrast, filters.vibrance, filters.sepia, filters.vignette, filters.brightness, filters.saturation, filters.exposure, filters.noise, filters.sharpen);
                                    applyFilters();
                                }
                            })
                            .catch(error => {
                                showPopup('Failed to upload image and filters.', 'error');
                                console.log(error);
                                setSliders(filters.contrast, filters.vibrance, filters.sepia, filters.vignette, filters.brightness, filters.saturation, filters.exposure, filters.noise, filters.sharpen);
                                applyFilters();
                            });
                    }, 'image/png');
                });

            }
        }
    });
});