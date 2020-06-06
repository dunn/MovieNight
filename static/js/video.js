/// <reference path="./both.js" />


async function initPlayer() {
  if (!flvjs.isSupported()) {
    console.warn('flvjs not supported');
    return;
  }

  const videoElement = document.querySelector("video");
  const flvPlayer = flvjs.createPlayer({
    type: "flv",
    url: "/live"
  });
  flvPlayer.attachMediaElement(videoElement);
  flvPlayer.load();
  flvPlayer.play();

  const overlay = document.querySelector('.overlay');
  overlay.onclick = () => {
    overlay.style.display = 'none';
    videoElement.muted = false;
  };

  videoElement.addEventListener('play', () => {
    document.querySelector(".poster").style.display = 'none';
  });

  videoElement.addEventListener('durationchange', () => {
    document.querySelector(".overlay").style.display = 'block';
    document.querySelector(".live").style.display = 'none';
  });

  videoElement.addEventListener('progress', () => {
    document.querySelector(".poster").style.display = 'none';
    document.querySelector(".live").style.display = 'block';
  });
}

window.addEventListener("load", initPlayer);
