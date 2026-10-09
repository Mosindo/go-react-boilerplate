import React from 'react';
import { FlatList, StyleSheet, View } from 'react-native';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { errorMessage } from '../../api/errors';
import { safetyApi } from '../../api/safety';
import type { BlockItem } from '../../api/types';
import { fr } from '../../i18n/fr';
import { qk } from '../../query/keys';
import { useBlocks } from '../../query/hooks';
import { Avatar, Button, EmptyState, ErrorState, LoadingView, Text } from '../../shared/ui';
import { toast } from '../../shared/toastStore';
import { bumpDiscoverEpoch } from '../../store/appState';
import { spacing, useTheme } from '../../theme';

export function BlockedScreen() {
  const qc = useQueryClient();
  const { colors } = useTheme();
  const { data, isPending, isError, error, refetch } = useBlocks();

  const unblock = useMutation({
    mutationFn: (userId: string) => safetyApi.unblock(userId),
    onSuccess: (_r, userId) => {
      qc.setQueryData<BlockItem[]>(qk.blocks, (old) => old?.filter((b) => b.user.id !== userId));
      bumpDiscoverEpoch();
      toast.success(fr.safety.unblocked);
    },
  });

  if (isPending) return <LoadingView />;
  if (isError) return <ErrorState message={errorMessage(error)} onRetry={() => void refetch()} />;

  return (
    <FlatList
      testID="blocked-list"
      data={data}
      keyExtractor={(b) => b.user.id}
      renderItem={({ item }) => (
        <View
          style={[styles.row, { borderColor: colors.border }]}
          testID={`blocked-${item.user.id}`}
        >
          <Avatar name={item.user.firstName} path={item.user.mainPhotoUrl} size={48} />
          <Text variant="bodyStrong" style={styles.name} numberOfLines={1}>
            {item.user.firstName}
          </Text>
          <Button
            label={fr.safety.unblock}
            variant="secondary"
            compact
            accessibilityLabel={`${fr.safety.unblock} ${item.user.firstName}`}
            loading={unblock.isPending && unblock.variables === item.user.id}
            onPress={() => unblock.mutate(item.user.id)}
            testID={`unblock-${item.user.id}`}
          />
        </View>
      )}
      ListEmptyComponent={
        <EmptyState
          icon="shield-checkmark-outline"
          title={fr.safety.blockedEmpty}
          testID="blocked-empty"
        />
      }
      contentContainerStyle={data.length === 0 ? styles.emptyContent : undefined}
    />
  );
}

const styles = StyleSheet.create({
  emptyContent: { flexGrow: 1 },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    borderBottomWidth: StyleSheet.hairlineWidth,
  },
  name: { flex: 1, marginHorizontal: spacing.md },
});
