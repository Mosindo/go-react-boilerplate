import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiError } from "../api/client";
import { deletePhoto, fetchPhotos, replacePhoto, reorderPhotos, uploadPhoto } from "../api/profile";
import type { Photo } from "../api/types";
import { pickPhoto } from "../lib/device";
import { confirmAction } from "../lib/platform";
import { showError } from "../shared/feedback";
import { keys } from "./queries";
import { useAuth } from "./useAuth";

/** Shared photo operations for onboarding and the "Mes photos" screen. */
export function usePhotoManager(enabled = true) {
  const queryClient = useQueryClient();
  const { refreshUser } = useAuth();
  const query = useQuery({ queryKey: keys.photos, queryFn: fetchPhotos, enabled });
  const photos = query.data ?? [];

  const mutation = useMutation({
    mutationFn: async (action: () => Promise<unknown>) => {
      await action();
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: keys.photos });
      void queryClient.invalidateQueries({ queryKey: keys.profile });
      void refreshUser();
    },
    onError: (err) => showError(err instanceof ApiError ? err.message : "Opération impossible sur cette photo.")
  });

  const add = async () => {
    const uri = await pickPhoto();
    if (uri) mutation.mutate(() => uploadPhoto(uri));
  };
  const replace = async (photo: Photo) => {
    const uri = await pickPhoto();
    if (uri) mutation.mutate(() => replacePhoto(photo.id, uri));
  };
  const remove = async (photo: Photo) => {
    if (await confirmAction("Supprimer cette photo ?", "Cette action est définitive.", "Supprimer"))
      mutation.mutate(() => deletePhoto(photo.id));
  };
  const move = (photo: Photo, to: number) => {
    const ids = photos.map((p) => p.id);
    const from = ids.indexOf(photo.id);
    if (from < 0 || to < 0 || to >= ids.length) return;
    ids.splice(to, 0, ids.splice(from, 1)[0] as string);
    mutation.mutate(() => reorderPhotos(ids));
  };

  return { query, photos, busy: mutation.isPending, add, replace, remove, move };
}
