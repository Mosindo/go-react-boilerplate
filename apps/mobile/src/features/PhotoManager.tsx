import React, { useState } from "react";
import { View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { deletePhoto, reorderPhotos, replacePhoto, uploadPhoto } from "../api";
import type { Photo as PhotoModel } from "../api/types";
import { errorMessage } from "../i18n/errorMessage";
import { t } from "../i18n/fr";
import { moveItem } from "../lib/format";
import { usePhotos } from "../queries";
import { queryKeys } from "../session/queryKeys";
import { createStyles } from "../theme/ThemeProvider";
import {
  EmptyState,
  ErrorState,
  Photo,
  ScalePressable,
  Skeleton,
  Text,
  useDialog,
  useToast
} from "../ui";
import { pickPhoto, type PickResult } from "./photoPicker";

export const MAX_PHOTOS = 6;

const useStyles = createStyles(({ colors, spacing, radius }) => ({
  grid: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  tileWrap: { width: "31.5%" },
  tile: { aspectRatio: 3 / 4, borderRadius: radius.md, overflow: "hidden" },
  add: {
    aspectRatio: 3 / 4,
    borderRadius: radius.md,
    borderWidth: 2,
    borderStyle: "dashed",
    borderColor: colors.border,
    alignItems: "center",
    justifyContent: "center",
    gap: spacing.xs,
    padding: spacing.sm
  },
  mainBadge: {
    position: "absolute",
    left: 6,
    top: 6,
    backgroundColor: colors.primary,
    borderRadius: radius.pill,
    paddingHorizontal: 8,
    paddingVertical: 2
  },
  busy: {
    position: "absolute",
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    backgroundColor: colors.overlay
  }
}));

type Props = {
  /** Refresh the account state (onboarding.completed) after each change. */
  refreshAccount: boolean;
};

export function PhotoManager({ refreshAccount }: Props) {
  const styles = useStyles();
  const toast = useToast();
  const dialog = useDialog();
  const queryClient = useQueryClient();
  const photos = usePhotos();
  const [busyId, setBusyId] = useState<string | "new" | null>(null);

  const afterChange = () => {
    void queryClient.invalidateQueries({ queryKey: queryKeys.photos });
    void queryClient.invalidateQueries({ queryKey: queryKeys.profile });
    if (refreshAccount) void queryClient.invalidateQueries({ queryKey: queryKeys.me });
  };

  const reportPickProblem = (result: PickResult) => {
    if (result.status === "denied") toast.error(t.photos.permissionDenied);
    else if (result.status === "too_small") toast.error(t.photos.tooSmall);
    else if (result.status === "error") toast.error(t.photos.pickFailed);
  };

  const add = useMutation({
    mutationFn: async () => {
      const picked = await pickPhoto();
      if (picked.status !== "ok") {
        reportPickProblem(picked);
        return false;
      }
      setBusyId("new");
      await uploadPhoto(picked.uri);
      return true;
    },
    onSuccess: (uploaded) => {
      if (uploaded) {
        toast.success(t.photos.uploaded);
        afterChange();
      }
    },
    onError: (error) => toast.error(errorMessage(error)),
    onSettled: () => setBusyId(null)
  });

  const replace = useMutation({
    mutationFn: async (photo: PhotoModel) => {
      const picked = await pickPhoto();
      if (picked.status !== "ok") {
        reportPickProblem(picked);
        return false;
      }
      setBusyId(photo.id);
      await replacePhoto(photo.id, picked.uri);
      return true;
    },
    onSuccess: (done) => {
      if (done) {
        toast.success(t.photos.replaced);
        afterChange();
      }
    },
    onError: (error) => toast.error(errorMessage(error)),
    onSettled: () => setBusyId(null)
  });

  const remove = useMutation({
    mutationFn: (photo: PhotoModel) => deletePhoto(photo.id),
    onSuccess: () => {
      toast.success(t.photos.deleted);
      afterChange();
    },
    onError: (error) => toast.error(errorMessage(error))
  });

  const reorder = useMutation({
    mutationFn: (next: PhotoModel[]) => reorderPhotos(next.map((photo) => photo.id)),
    onMutate: async (next) => {
      await queryClient.cancelQueries({ queryKey: queryKeys.photos });
      const previous = queryClient.getQueryData<PhotoModel[]>(queryKeys.photos);
      queryClient.setQueryData(queryKeys.photos, next);
      return { previous };
    },
    onError: (error, _next, context) => {
      if (context?.previous) queryClient.setQueryData(queryKeys.photos, context.previous);
      toast.error(errorMessage(error));
    },
    onSuccess: () => {
      toast.success(t.photos.reordered);
      afterChange();
    }
  });

  if (photos.isPending) {
    return (
      <View style={styles.grid}>
        {[0, 1, 2].map((key) => (
          <View key={key} style={styles.tileWrap}>
            <Skeleton height={140} radius={16} />
          </View>
        ))}
      </View>
    );
  }
  if (photos.isError) {
    return <ErrorState onRetry={() => photos.refetch()} title={t.profile.loadError} />;
  }

  const list = [...photos.data].sort((a, b) => a.position - b.position);

  const openOptions = (photo: PhotoModel, index: number) => {
    const actions = [];
    if (index > 0) {
      actions.push({
        label: t.photos.makeMain,
        onPress: () => reorder.mutate(moveItem(list, index, 0))
      });
      actions.push({
        label: t.photos.moveLeft,
        onPress: () => reorder.mutate(moveItem(list, index, index - 1))
      });
    }
    if (index < list.length - 1) {
      actions.push({
        label: t.photos.moveRight,
        onPress: () => reorder.mutate(moveItem(list, index, index + 1))
      });
    }
    actions.push({ label: t.photos.replace, onPress: () => replace.mutate(photo) });
    actions.push({
      label: t.photos.delete,
      destructive: true,
      onPress: () => {
        if (list.length <= 1) {
          toast.error(t.photos.lastPhoto);
          return;
        }
        dialog.confirm({
          title: t.photos.deleteConfirmTitle,
          message: t.photos.deleteConfirmText,
          confirmLabel: t.photos.delete,
          destructive: true,
          onConfirm: () => remove.mutate(photo)
        });
      }
    });
    dialog.showSheet({ title: t.photos.slot(index + 1), actions });
  };

  return (
    <View>
      {list.length === 0 ? (
        <EmptyState glyph="+" message={t.photos.text} title={t.photos.add} />
      ) : null}
      <View style={styles.grid}>
        {list.map((photo, index) => (
          <View key={photo.id} style={styles.tileWrap}>
            <ScalePressable
              accessibilityHint={t.photos.options}
              accessibilityLabel={`${t.photos.slot(index + 1)}${index === 0 ? `, ${t.photos.main}` : ""}`}
              accessibilityRole="button"
              onPress={() => openOptions(photo, index)}
              testID={`photo-tile-${index}`}
            >
              <View style={styles.tile}>
                <Photo path={photo.thumbUrl} style={styles.tile} />
                {index === 0 ? (
                  <View style={styles.mainBadge}>
                    <Text variant="caption" tone="onPrimary">
                      {t.photos.main}
                    </Text>
                  </View>
                ) : null}
                {busyId === photo.id ? <View style={styles.busy} /> : null}
              </View>
            </ScalePressable>
          </View>
        ))}
        {list.length < MAX_PHOTOS ? (
          <View style={styles.tileWrap}>
            <ScalePressable
              accessibilityLabel={t.photos.add}
              accessibilityRole="button"
              disabled={add.isPending}
              onPress={() => add.mutate()}
              testID="photo-add-button"
            >
              <View style={styles.add}>
                <Text variant="title" tone="primary">
                  {add.isPending ? "…" : "+"}
                </Text>
                <Text variant="caption" center tone="muted">
                  {t.photos.add}
                </Text>
              </View>
            </ScalePressable>
          </View>
        ) : null}
      </View>
    </View>
  );
}
