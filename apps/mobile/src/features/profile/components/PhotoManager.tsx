import { useState } from 'react';
import { ActivityIndicator, Pressable, View } from 'react-native';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { mediaUrl, photosApi, queryKeys } from '../../../api';
import type { OwnProfile, Photo } from '../../../api';
import { fr } from '../../../i18n';
import { MAX_PHOTOS } from '../../../lib/validation';
import { pickImage } from '../../../lib/pickImage';
import type { PickResult } from '../../../lib/pickImage';
import { useErrorToast } from '../../../lib/useErrorToast';
import { useStyles, useTheme } from '../../../theme';
import { ActionSheet, ConfirmDialog, RemoteImage, Text, icons, useToast } from '../../../ui';
import type { SheetAction } from '../../../ui';
import { useAuth } from '../../auth/AuthContext';
import { movePhoto } from '../photoOrder';

interface PhotoManagerProps {
  photos: Photo[];
}

export function PhotoManager({ photos }: PhotoManagerProps) {
  const { colors } = useTheme();
  const qc = useQueryClient();
  const toast = useToast();
  const onError = useErrorToast();
  const { refreshUser } = useAuth();
  const [selected, setSelected] = useState<Photo | null>(null);
  const [toDelete, setToDelete] = useState<Photo | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);

  const ordered = [...photos].sort((a, b) => a.position - b.position);
  const styles = useStyles((t) => ({
    grid: { flexDirection: 'row', flexWrap: 'wrap', gap: t.spacing.sm },
    slot: { width: '31.5%', aspectRatio: 3 / 4, borderRadius: t.radius.md, overflow: 'hidden' },
    empty: {
      borderWidth: 2,
      borderStyle: 'dashed',
      borderColor: t.colors.border,
      alignItems: 'center',
      justifyContent: 'center',
      backgroundColor: t.colors.surface,
    },
    fill: { flex: 1 },
    primaryTag: {
      position: 'absolute',
      left: 6,
      bottom: 6,
      backgroundColor: t.colors.primary,
      borderRadius: t.radius.pill,
      paddingHorizontal: 8,
      paddingVertical: 2,
    },
    busy: {
      position: 'absolute',
      top: 0,
      left: 0,
      right: 0,
      bottom: 0,
      backgroundColor: t.colors.scrim,
      alignItems: 'center',
      justifyContent: 'center',
    },
    gap: { gap: t.spacing.md },
  }));

  const syncAfterChange = async () => {
    await qc.invalidateQueries({ queryKey: queryKeys.ownProfile });
    await refreshUser();
  };

  const report = (result: PickResult): result is Extract<PickResult, { status: 'picked' }> => {
    if (result.status === 'picked') return true;
    if (result.status === 'too_large') toast.error(fr.photos.tooLarge);
    if (result.status === 'unavailable') toast.error(fr.photos.pickerUnavailable);
    return false;
  };

  const add = useMutation({
    mutationFn: async () => {
      const result = await pickImage();
      if (!report(result)) return false;
      await photosApi.upload(result.image);
      return true;
    },
    onSuccess: async (uploaded) => {
      if (uploaded) {
        await syncAfterChange();
        toast.success(fr.photos.added);
      }
    },
    onError,
  });

  const replace = useMutation({
    mutationFn: async (photo: Photo) => {
      const result = await pickImage();
      if (!report(result)) return false;
      setBusyId(photo.id);
      await photosApi.replace(photo.id, result.image);
      return true;
    },
    onSuccess: async (done) => {
      if (done) {
        await syncAfterChange();
        toast.success(fr.photos.replaced);
      }
    },
    onError,
    onSettled: () => setBusyId(null),
  });

  const remove = useMutation({
    mutationFn: (photo: Photo) => photosApi.remove(photo.id),
    onMutate: (photo) => setBusyId(photo.id),
    onSuccess: async () => {
      setToDelete(null);
      await syncAfterChange();
      toast.success(fr.photos.deleted);
    },
    onError,
    onSettled: () => setBusyId(null),
  });

  const reorder = useMutation({
    mutationFn: (next: Photo[]) => photosApi.reorder(next.map((p) => p.id)),
    onMutate: async (next) => {
      await qc.cancelQueries({ queryKey: queryKeys.ownProfile });
      const previous = qc.getQueryData<OwnProfile>(queryKeys.ownProfile);
      qc.setQueryData<OwnProfile>(queryKeys.ownProfile, (old) =>
        old ? { ...old, photos: next } : old,
      );
      return { previous };
    },
    onError: (error, _next, context) => {
      if (context?.previous) qc.setQueryData(queryKeys.ownProfile, context.previous);
      onError(error);
    },
    onSettled: () => void syncAfterChange(),
  });

  const actionsFor = (photo: Photo): SheetAction[] => {
    const index = ordered.findIndex((p) => p.id === photo.id);
    const actions: SheetAction[] = [];
    if (index > 0) {
      actions.push({
        label: fr.photos.makePrimary,
        onPress: () => reorder.mutate(movePhoto(ordered, index, 0)),
      });
      actions.push({
        label: fr.photos.moveEarlier,
        onPress: () => reorder.mutate(movePhoto(ordered, index, index - 1)),
      });
    }
    if (index < ordered.length - 1) {
      actions.push({
        label: fr.photos.moveLater,
        onPress: () => reorder.mutate(movePhoto(ordered, index, index + 1)),
      });
    }
    actions.push({ label: fr.photos.replace, onPress: () => replace.mutate(photo) });
    actions.push({
      label: fr.photos.delete,
      variant: 'danger',
      onPress: () => (ordered.length <= 1 ? toast.error(fr.photos.keepOne) : setToDelete(photo)),
    });
    return actions;
  };

  const emptySlots = Math.max(0, MAX_PHOTOS - ordered.length);

  return (
    <View style={styles.gap}>
      <View style={styles.grid}>
        {ordered.map((photo, index) => (
          <Pressable
            key={photo.id}
            style={styles.slot}
            accessibilityRole="button"
            accessibilityLabel={fr.photos.photoLabel(index + 1, photo.isPrimary)}
            accessibilityHint={fr.photos.photoHint}
            onPress={() => setSelected(photo)}
          >
            <RemoteImage uri={mediaUrl(photo.url)} style={styles.fill} />
            {photo.isPrimary ? (
              <View style={styles.primaryTag}>
                <Text variant="caption" color="textOnPrimary" weight="700" style={{ fontSize: 11 }}>
                  {fr.photos.primary}
                </Text>
              </View>
            ) : null}
            {busyId === photo.id ? (
              <View style={styles.busy}>
                <ActivityIndicator color="#FFFFFF" />
              </View>
            ) : null}
          </Pressable>
        ))}
        {Array.from({ length: emptySlots }, (_, i) => {
          const isNext = i === 0;
          return (
            <Pressable
              key={`empty-${i}`}
              style={[styles.slot, styles.empty]}
              accessibilityRole="button"
              accessibilityLabel={fr.photos.add}
              accessibilityState={{
                disabled: !isNext || add.isPending,
                busy: isNext && add.isPending,
              }}
              disabled={!isNext || add.isPending}
              onPress={() => add.mutate()}
              testID={isNext ? 'photo-add' : undefined}
            >
              {isNext && add.isPending ? (
                <ActivityIndicator color={colors.primary} />
              ) : (
                <Text
                  color={isNext ? 'primary' : 'textMuted'}
                  style={{ fontSize: 30, opacity: isNext ? 1 : 0.4 }}
                >
                  {icons.plus}
                </Text>
              )}
            </Pressable>
          );
        })}
      </View>
      <Text variant="caption" color="textMuted">
        {fr.photos.helper(ordered.length, MAX_PHOTOS)}
      </Text>

      <ActionSheet
        visible={selected !== null}
        onClose={() => setSelected(null)}
        title={fr.photos.sheetTitle}
        actions={selected ? actionsFor(selected) : []}
      />
      <ConfirmDialog
        visible={toDelete !== null}
        title={fr.photos.deleteTitle}
        message={fr.photos.deleteMessage}
        confirmLabel={fr.photos.delete}
        destructive
        loading={remove.isPending}
        onConfirm={() => toDelete && remove.mutate(toDelete)}
        onCancel={() => setToDelete(null)}
      />
    </View>
  );
}
