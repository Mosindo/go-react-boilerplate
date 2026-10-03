import React from "react";
import { PhotoManager } from "../components/PhotoManager";
import { Screen } from "../components/Screen";

export default function PhotosScreen() {
  return (
    <Screen testID="photos-screen">
      <PhotoManager />
    </Screen>
  );
}
