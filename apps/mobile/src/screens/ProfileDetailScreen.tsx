import React, { useLayoutEffect, useState } from 'react';
import { StyleSheet, View } from 'react-native';
import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { errorMessage, isApiError } from '../api/errors';
import { announceSwipe, performSwipe, toSwipeTarget } from '../features/swipe';
import { fr } from '../i18n/fr';
import type { MainStackParamList } from '../navigation/types';
import { useProfileById } from '../query/hooks';
import { ProfileView } from '../shared/ProfileView';
import { UserActionsMenu } from '../shared/UserActionsMenu';
import { EmptyState, ErrorState, IconButton, LoadingView, RoundButton, Screen } from '../shared/ui';
import type { SwipeAction } from '../api/types';
import { spacing, useTheme } from '../theme';

export function ProfileDetailScreen({
  route,
  navigation,
}: NativeStackScreenProps<MainStackParamList, 'ProfileDetail'>) {
  const { userId, mode = 'view' } = route.params;
  const qc = useQueryClient();
  const { colors } = useTheme();
  const { data, isPending, isError, error, refetch } = useProfileById(userId);
  const [menu, setMenu] = useState(false);

  useLayoutEffect(() => {
    navigation.setOptions({
      title: data?.firstName ?? '',
      headerRight: () =>
        data ? (
          <IconButton
            icon="ellipsis-horizontal"
            label={fr.messages.menu}
            onPress={() => setMenu(true)}
            testID="profile-menu"
          />
        ) : null,
    });
  }, [navigation, data]);

  const swipe = useMutation({
    mutationFn: async (action: SwipeAction) => {
      await performSwipe(qc, toSwipeTarget(data!), action);
      return action;
    },
    onSuccess: (action) => {
      announceSwipe(userId, action);
      navigation.goBack();
    },
  });

  if (isPending) return <LoadingView />;
  if (isError) {
    return isApiError(error, 'not_found') ? (
      <EmptyState
        icon="person-remove-outline"
        title={fr.discover.profileUnavailable}
        actionLabel={fr.common.back}
        onAction={() => navigation.goBack()}
      />
    ) : (
      <ErrorState message={errorMessage(error)} onRetry={() => void refetch()} />
    );
  }

  const footer =
    mode === 'swipe' ? (
      <View style={styles.actions}>
        <RoundButton
          testID="detail-pass"
          icon="close"
          label={fr.discover.pass}
          color={colors.textMuted}
          disabled={swipe.isPending}
          onPress={() => swipe.mutate('pass')}
        />
        <RoundButton
          testID="detail-like"
          icon="heart"
          label={fr.discover.like}
          color={colors.primary}
          filled
          disabled={swipe.isPending}
          onPress={() => swipe.mutate('like')}
        />
      </View>
    ) : undefined;

  return (
    <Screen
      scroll
      padded={false}
      contentStyle={styles.content}
      footer={footer}
      testID="profile-detail-screen"
    >
      <ProfileView profile={data} />
      <UserActionsMenu
        visible={menu}
        onClose={() => setMenu(false)}
        user={{ id: data.id, firstName: data.firstName }}
        onBlocked={() => navigation.popToTop()}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  content: { padding: spacing.lg },
  actions: { flexDirection: 'row', justifyContent: 'center', gap: spacing.xxl },
});
