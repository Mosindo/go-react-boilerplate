import { useState } from "react";
import * as Location from "expo-location";
import { useQueryClient } from "@tanstack/react-query";
import { profileApi } from "../../api/endpoints";
import { errorMessage } from "../../lib/format";
import { keys } from "../../providers/queryClient";
import { showToast } from "../../ui";

/**
 * Requests foreground location once, on explicit user action only. A coarse
 * fix is enough: the value is rounded on the device and snapped again to a
 * ~1 km grid by the server, which never stores the exact position.
 */
export function useLocationUpdate() {
  const queryClient = useQueryClient();
  const [pending, setPending] = useState(false);

  const update = async (): Promise<boolean> => {
    setPending(true);
    try {
      const permission = await Location.requestForegroundPermissionsAsync();
      if (!permission.granted) {
        showToast("Sans accès à la position, les distances ne seront pas affichées.", "error");
        return false;
      }
      const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Low });
      const round = (v: number) => Math.round(v * 100) / 100;
      await profileApi.setLocation(round(position.coords.latitude), round(position.coords.longitude));
      await queryClient.invalidateQueries({ queryKey: keys.profile });
      void queryClient.invalidateQueries({ queryKey: keys.discovery });
      showToast("Position approximative mise à jour.", "success");
      return true;
    } catch (error) {
      showToast(errorMessage(error, "Impossible d’obtenir ta position."), "error");
      return false;
    } finally {
      setPending(false);
    }
  };

  return { update, pending };
}
