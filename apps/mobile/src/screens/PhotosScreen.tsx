import { PhotoManager } from "../components/PhotoManager";
import { Screen } from "../components/ui";

export function PhotosScreen() {
  return (
    <Screen edges={["bottom"]} scroll>
      <PhotoManager />
    </Screen>
  );
}
