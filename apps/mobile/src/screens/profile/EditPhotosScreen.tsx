import React from 'react';
import { PhotoManager } from '../../shared/forms/PhotoManager';
import { Screen } from '../../shared/ui';

export function EditPhotosScreen() {
  return (
    <Screen scroll testID="edit-photos-screen">
      <PhotoManager />
    </Screen>
  );
}
