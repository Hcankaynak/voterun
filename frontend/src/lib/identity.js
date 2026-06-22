// A stable per-browser identity so votes can be attributed and toggled,
// and so we can pre-fill the author name. No accounts required.

const ID_KEY = "voterun:voterId";
const NAME_KEY = "voterun:name";

function randomId() {
  if (crypto?.randomUUID) return crypto.randomUUID();
  return `voter-${Math.random().toString(36).slice(2)}-${Date.now()}`;
}

export function getVoterId() {
  let id = localStorage.getItem(ID_KEY);
  if (!id) {
    id = randomId();
    localStorage.setItem(ID_KEY, id);
  }
  return id;
}

export function getName() {
  return localStorage.getItem(NAME_KEY) || "";
}

export function setName(name) {
  localStorage.setItem(NAME_KEY, name);
}
