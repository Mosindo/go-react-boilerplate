import React, { useState } from 'react';
import { Pressable, StyleSheet, View } from 'react-native';
import * as ImagePicker from 'expo-image-picker';
import { Ionicons } from '@expo/vector-icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { profileApi } from '../../api/profile';
import type { Photo } from '../../api/types';
import { fr } from '../../i18n/fr';
import { qk } from '../../query/keys';
import { radius, spacing, useTheme } from '../../theme';
import { prepareImageForUpload } from '../../utils/image';
import { toast } from '../toastStore';
import { ActionSheet, AuthImage, ConfirmDialog, ErrorState, LoadingView, Text } from '../ui';

const SLOTS = 6;

/** Photo grid: add (picker + client-side compression), set main, reorder, delete. */
export function PhotoManager() {
  const qc = useQueryClient();
  const { colors } = useTheme();
  const { data, isPending, isError, refetch } = useQuery({
    queryKey: qk.myPhotos,
    queryFn: profileApi.listPhotos,
  });
  const [menuIndex, setMenuIndex] = useState<number | null>(null);
  const [deleteIndex, setDeleteIndex] = useState<number | null>(null);

  const photos: Photo[] = data ?? [];

  const sync = (next: Photo[]) => {
    qc.setQueryData(qk.myPhotos, next);
    void qc.invalidateQueries({ queryKey: qk.me });
    void qc.invalidateQueries({ queryKey: qk.myProfile });
  };

  const upload = useMutation({
    mutationFn: async () => {
      const picked = await ImagePicker.launchImageLibraryAsync({
        mediaTypes: ['images'],
        allowsMultipleSelection: false,
        quality: 1,
      });
      const asset = picked.canceled ? null : picked.assets[0];
      if (!asset) return null;
      const uri = await prepareImageForUpload(asset.uri, asset.width, asset.height);
      return profileApi.uploadPhoto(uri);
    },
    onSuccess: (photo) => {
      if (photo) sync([...(qc.getQueryData<Photo[]>(qk.myPhotos) ?? []), photo]);
    },
  });

  const reorder = useMutation({
    mutationFn: (ids: string[]) => profileApi.reorderPhotos(ids),
    onMutate: async (ids) => {
      const previous = qc.getQueryData<Photo[]>(qk.myPhotos) ?? [];
      qc.setQueryData(
        qk.myPhotos,
        ids.map((id, position) => ({ ...previous.find((p) => p.id === id)!, position })),
      );
      return { previous };
    },
    onError: (_e, _v, ctx) => ctx && qc.setQueryData(qk.myPhotos, ctx.previous),
    onSuccess: sync,
  });

  const makeMain = useMutation({
    mutationFn: (id: string) => profileApi.setMainPhoto(id),
    onSuccess: sync,
  });

  const remove = useMutation({
    mutationFn: (id: string) => profileApi.deletePhoto(id),
    onSuccess: (_r, id) => {
      sync((qc.getQueryData<Photo[]>(qk.myPhotos) ?? []).filter((p) => p.id !== id));
      toast.success(fr.common.saved);
    },
    onSettled: () => setDeleteIndex(null),
  });

  const move = (index: number, delta: -1 | 1) => {
    const target = index + delta;
    if (target < 0 || target >= photos.length) return;
    const ids = photos.map((p) => p.id);
    [ids[index], ids[target]] = [ids[target]!, ids[index]!];
    reorder.mutate(ids);
  };

  if (isPending) return <LoadingView />;
  if (isError) return <ErrorState onRetry={() => void refetch()} />;

  const menuPhoto = menuIndex !== null ? photos[menuIndex] : undefined;
  const menuItems = menuPhoto
    ? [
        ...(menuIndex !== 0
          ? [
              {
                key: 'main',
                label: fr.photos.makeMain,
                icon: 'star-outline' as const,
                onPress: () => makeMain.mutate(menuPhoto.id),
              },
            ]
          : []),
        ...(menuIndex! > 0
          ? [
              {
                key: 'left',
                label: fr.photos.moveLeft,
                icon: 'arrow-back' as const,
                onPress: () => move(menuIndex!, -1),
              },
            ]
          : []),
        ...(menuIndex! < photos.length - 1
          ? [
              {
                key: 'right',
                label: fr.photos.moveRight,
                icon: 'arrow-forward' as const,
                onPress: () => move(menuIndex!, 1),
              },
            ]
          : []),
        {
          key: 'delete',
          label: fr.photos.remove,
          icon: 'trash-outline' as const,
          destructive: true,
          onPress: () => setDeleteIndex(menuIndex),
        },
      ]
    : [];

  return (
    <View testID="photo-manager">
      <Text variant="label" tone="muted" style={styles.count}>
        {fr.photos.count(photos.length)}
      </Text>
      <View style={styles.grid}>
        {Array.from({ length: SLOTS }, (_, i) => {
          const photo = photos[i];
          if (photo) {
            return (
              <View key={photo.id} style={styles.cell}>
                <Pressable
                  testID={`photo-tile-${i}`}
                  accessibilityRole="button"
                  accessibilityLabel={`${fr.photos.photoOf(fr.app.name, i + 1, photos.length)}${i === 0 ? `, ${fr.photos.main}` : ''}`}
                  onPress={() => setMenuIndex(i)}
                  style={[styles.tile, { borderColor: i === 0 ? colors.primary : colors.border }]}
                >
                  <AuthImage path={photo.url} style={styles.img} />
                  {i === 0 ? (
                    <View style={[styles.mainBadge, { backgroundColor: colors.primary }]}>
                      <Text
                        variant="caption"
                        style={{ color: colors.onPrimary, fontWeight: '700' }}
                      >
                        {fr.photos.main}
                      </Text>
                    </View>
                  ) : null}
                </Pressable>
                <Pressable
                  testID={`photo-delete-${i}`}
                  accessibilityRole="button"
                  accessibilityLabel={`${fr.photos.remove} ${i + 1}`}
                  onPress={() => setDeleteIndex(i)}
                  hitSlop={8}
                  style={[styles.del, { backgroundColor: colors.scrim }]}
                >
                  <Ionicons name="close" size={16} color="#FFFFFF" />
                </Pressable>
              </View>
            );
          }
          const isNext = i === photos.length;
          return (
            <View key={`slot-${i}`} style={styles.cell}>
              <Pressable
                testID={isNext ? 'photo-add' : `photo-slot-${i}`}
                accessibilityRole="button"
                accessibilityLabel={fr.photos.addAccessibility}
                accessibilityState={{
                  disabled: !isNext || upload.isPending,
                  busy: upload.isPending && isNext,
                }}
                disabled={!isNext || upload.isPending}
                onPress={() => upload.mutate()}
                style={[
                  styles.tile,
                  styles.empty,
                  { borderColor: colors.border, backgroundColor: colors.surfaceAlt },
                  !isNext ? { opacity: 0.5 } : null,
                ]}
              >
                <Ionicons
                  name={upload.isPending && isNext ? 'hourglass-outline' : 'add'}
                  size={28}
                  color={colors.textMuted}
                />
                {isNext ? (
                  <Text variant="caption" tone="muted" center>
                    {upload.isPending ? fr.photos.uploading : fr.photos.add}
                  </Text>
                ) : null}
              </Pressable>
            </View>
          );
        })}
      </View>

      <ActionSheet
        visible={menuIndex !== null && !!menuPhoto}
        onClose={() => setMenuIndex(null)}
        items={menuItems}
        testID="photo-menu"
      />
      <ConfirmDialog
        visible={deleteIndex !== null}
        title={fr.photos.removeConfirmTitle}
        body={fr.photos.removeConfirmBody}
        confirmLabel={fr.common.delete}
        destructive
        loading={remove.isPending}
        onCancel={() => setDeleteIndex(null)}
        onConfirm={() => {
          const p = deleteIndex !== null ? photos[deleteIndex] : undefined;
          if (p) remove.mutate(p.id);
        }}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  count: { marginBottom: spacing.sm },
  grid: { flexDirection: 'row', flexWrap: 'wrap', marginHorizontal: -spacing.xs },
  cell: { width: '33.333%', padding: spacing.xs },
  tile: {
    aspectRatio: 3 / 4,
    borderRadius: radius.md,
    borderWidth: 2,
    overflow: 'hidden',
  },
  empty: { alignItems: 'center', justifyContent: 'center', borderStyle: 'dashed' },
  img: { width: '100%', height: '100%' },
  mainBadge: {
    position: 'absolute',
    left: 6,
    bottom: 6,
    paddingHorizontal: 8,
    paddingVertical: 2,
    borderRadius: radius.pill,
  },
  del: {
    position: 'absolute',
    top: spacing.sm + 2,
    right: spacing.sm + 2,
    width: 24,
    height: 24,
    borderRadius: 12,
    alignItems: 'center',
    justifyContent: 'center',
  },
});
