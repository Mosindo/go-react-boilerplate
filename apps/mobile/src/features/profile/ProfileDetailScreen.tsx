import { useLayoutEffect, useState } from 'react';
import { ScrollView, View } from 'react-native';
import { useQuery } from '@tanstack/react-query';
import { errorMessage, isNotFound, profileApi, queryKeys } from '../../api';
import { fr } from '../../i18n';
import type { RootScreenProps } from '../../navigation/types';
import { useStyles } from '../../theme';
import { EmptyState, ErrorState, IconButton, Loading, Screen, icons } from '../../ui';
import { useDeck } from '../discover/DeckContext';
import { useRespondToLike } from '../likes/hooks';
import { UserActionsSheet } from '../safety/UserActionsSheet';
import { ProfileView } from './components/ProfileView';

export function ProfileDetailScreen({ navigation, route }: RootScreenProps<'ProfileDetail'>) {
  const { userId, profile: preloaded, source } = route.params;
  const deck = useDeck();
  const respondToLike = useRespondToLike();
  const [menuOpen, setMenuOpen] = useState(false);
  const query = useQuery({
    queryKey: queryKeys.profile(userId),
    queryFn: () => profileApi.publicProfile(userId),
    initialData: preloaded,
    staleTime: 60_000,
    retry: (count, error) => !isNotFound(error) && count < 2,
  });
  const styles = useStyles((t) => ({
    scroll: { paddingVertical: t.spacing.lg, paddingBottom: t.spacing.xxxl },
    actions: {
      flexDirection: 'row',
      justifyContent: 'center',
      gap: t.spacing.xxl,
      paddingVertical: t.spacing.md,
    },
  }));

  useLayoutEffect(() => {
    navigation.setOptions({
      headerRight: () => (
        <IconButton
          glyph={icons.more}
          accessibilityLabel={fr.profile.moreActions}
          onPress={() => setMenuOpen(true)}
        />
      ),
    });
  }, [navigation]);

  const data = query.data;
  const canSwipe = source === 'discover' || source === 'likes';

  const swipe = (action: 'like' | 'pass') => {
    if (!data) return;
    if (source === 'discover') deck.act(data, action);
    else respondToLike(data, action);
    navigation.goBack();
  };

  if (!data && query.isPending) return <Loading />;
  if (!data) {
    return (
      <Screen edges={['bottom']}>
        {isNotFound(query.error) ? (
          <EmptyState
            glyph={icons.lock}
            title={fr.profile.unavailableTitle}
            message={fr.profile.unavailableMessage}
            actionLabel={fr.common.back}
            onAction={() => navigation.goBack()}
          />
        ) : (
          <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />
        )}
      </Screen>
    );
  }

  return (
    <Screen edges={['bottom']} padded={false}>
      <ScrollView contentContainerStyle={[styles.scroll, { paddingHorizontal: 16 }]}>
        <ProfileView profile={data} />
      </ScrollView>
      {canSwipe ? (
        <View style={styles.actions}>
          <IconButton
            glyph={icons.close}
            accessibilityLabel={fr.discover.pass}
            size={64}
            tone="outline"
            color="danger"
            onPress={() => swipe('pass')}
          />
          <IconButton
            glyph={icons.heart}
            accessibilityLabel={fr.discover.like}
            size={64}
            tone="primary"
            onPress={() => swipe('like')}
          />
        </View>
      ) : null}
      <UserActionsSheet
        visible={menuOpen}
        onClose={() => setMenuOpen(false)}
        target={{ userId, firstName: data.firstName }}
        onGone={() => navigation.goBack()}
      />
    </Screen>
  );
}
