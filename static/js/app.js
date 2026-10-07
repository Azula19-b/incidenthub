const menuButton = document.querySelector('#menuButton');
const sidebar = document.querySelector('#sidebar');

menuButton?.addEventListener('click', () => {
  sidebar?.classList.toggle('open');
});
