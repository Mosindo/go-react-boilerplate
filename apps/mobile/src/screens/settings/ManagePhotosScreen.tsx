import React from "react";
import { PhotoManager } from "../../features/PhotoManager";
import { t } from "../../i18n/fr";
import { Screen, Text } from "../../ui";

export default function ManagePhotosScreen() {
  return (
    <Screen scroll testID="manage-photos-screen">
      <Text tone="muted">{t.photos.text}</Text>
      <PhotoManager refreshAccount />
    </Screen>
  );
}
