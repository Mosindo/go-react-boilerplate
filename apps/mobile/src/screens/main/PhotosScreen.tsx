import React from "react";
import { PhotoGrid } from "../../components/PhotoGrid";
import { Screen } from "../../components/Screen";

export default function PhotosScreen() {
  return (
    <Screen edges={["bottom"]} scroll>
      <PhotoGrid />
    </Screen>
  );
}
