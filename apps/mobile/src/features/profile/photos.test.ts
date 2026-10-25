import type { Photo, ProfileState } from "../../api";
import { addPhoto, movePhotoId, removePhoto, replacePhoto } from "./photoCache";
import { toUploadFile } from "./photoFile";
import { profileToInput } from "./profileInput";

const photo = (id: string, position: number): Photo => ({
  id,
  position,
  isPrimary: position === 0,
  url: `/photos/${id}/image`,
  thumbUrl: `/photos/${id}/image?size=thumb`
});

const state = (photos: Photo[]): ProfileState => ({
  profile: {
    userId: "me",
    firstName: "Alex",
    birthDate: "1990-01-02",
    age: 36,
    gender: "woman",
    bio: "Hello",
    city: "Paris",
    hasLocation: true,
    discoverable: true,
    showDistance: false,
    interests: [{ slug: "hiking", label: "Randonnée" }],
    photos
  },
  preferences: null,
  complete: false
});

describe("photo cache", () => {
  it("adds, compacts positions on removal and keeps the first photo primary", () => {
    const added = addPhoto(state([photo("a", 0)]), photo("b", 1));
    expect(added?.profile?.photos.map((p) => p.id)).toEqual(["a", "b"]);
    const removed = removePhoto(added, "a");
    expect(removed?.profile?.photos).toEqual([{ ...photo("b", 0), isPrimary: true }]);
  });

  it("replaces a photo by its old id because the API issues a new one", () => {
    const next = replacePhoto(state([photo("a", 0), photo("b", 1)]), "a", photo("a2", 0));
    expect(next?.profile?.photos.map((p) => p.id)).toEqual(["a2", "b"]);
  });

  it("computes reorder payloads", () => {
    const photos = [photo("a", 0), photo("b", 1), photo("c", 2)];
    expect(movePhotoId(photos, "c", -2)).toEqual(["c", "a", "b"]);
    expect(movePhotoId(photos, "a", 1)).toEqual(["b", "a", "c"]);
    expect(movePhotoId(photos, "a", -1)).toBeNull();
  });
});

describe("upload files and profile input", () => {
  it("names uploads by their real type", () => {
    expect(toUploadFile({ uri: "file:///x/IMG_1.HEIC", fileName: "IMG_1.HEIC", mimeType: "image/heic" })).toEqual({
      uri: "file:///x/IMG_1.HEIC",
      name: "IMG_1.jpg",
      type: "image/jpeg"
    });
    expect(toUploadFile({ uri: "file:///x/shot.png", mimeType: "image/png" })).toMatchObject({
      name: "photo.png",
      type: "image/png"
    });
  });

  it("rebuilds a full PUT /profile body from the current profile", () => {
    const profile = state([]).profile;
    if (!profile) throw new Error("fixture");
    expect(profileToInput(profile, { showDistance: true })).toEqual({
      firstName: "Alex",
      birthDate: "1990-01-02",
      gender: "woman",
      bio: "Hello",
      interests: ["hiking"],
      discoverable: true,
      showDistance: true
    });
  });
});
