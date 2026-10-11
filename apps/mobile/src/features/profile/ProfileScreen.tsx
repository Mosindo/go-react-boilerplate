import { useState } from 'react';
import { RefreshControl, View } from 'react-native';
import { errorMessage } from '../../api';
import type { PublicProfile } from '../../api';
import { fr } from '../../i18n';
import type { TabScreenProps } from '../../navigation/types';
import { useStyles, useTheme } from '../../theme';
import {
  Badge,
  Card,
  ConfirmDialog,
  Divider,
  ErrorState,
  ListRow,
  Loading,
  Screen,
  Text,
  useToast,
} from '../../ui';
import { useAuth } from '../auth/AuthContext';
import { useNotifications } from '../notifications/hooks';
import { ProfileView } from './components/ProfileView';
import { useOwnProfile } from './hooks';

export function ProfileScreen({ navigation }: TabScreenProps<'Profile'>) {
  const { colors } = useTheme();
  const toast = useToast();
  const { logout } = useAuth();
  const profile = useOwnProfile();
  const { unreadCount } = useNotifications();
  const [confirmLogout, setConfirmLogout] = useState(false);
  const [loggingOut, setLoggingOut] = useState(false);
  const styles = useStyles((t) => ({
    header: { paddingVertical: t.spacing.md },
    group: { gap: t.spacing.sm },
  }));

  if (profile.isPending) return <Loading />;
  if (profile.isError) {
    return (
      <Screen edges={['top']}>
        <ErrorState message={errorMessage(profile.error)} onRetry={() => void profile.refetch()} />
      </Screen>
    );
  }

  const own = profile.data;
  const preview: PublicProfile = { ...own, distanceKm: null };

  return (
    <Screen
      scroll
      edges={['top']}
      refreshControl={
        <RefreshControl
          refreshing={profile.isRefetching}
          onRefresh={() => void profile.refetch()}
          tintColor={colors.primary}
          colors={[colors.primary]}
        />
      }
    >
      <View style={styles.header}>
        <Text variant="title">{fr.profile.title}</Text>
        <Text color="textMuted">{fr.profile.previewHint}</Text>
      </View>
      <ProfileView profile={preview} />

      <View style={styles.group}>
        <Text variant="label" color="textMuted">
          {fr.profile.sectionProfile}
        </Text>
        <Card padded={false}>
          <ListRow
            title={fr.profile.edit}
            onPress={() => navigation.navigate('EditProfile')}
            testID="profile-edit"
          />
          <Divider />
          <ListRow
            title={fr.profile.managePhotos}
            subtitle={fr.photos.count(own.photos.length)}
            onPress={() => navigation.navigate('Photos')}
          />
          <Divider />
          <ListRow
            title={fr.profile.preferences}
            onPress={() => navigation.navigate('Preferences')}
          />
          <Divider />
          <ListRow title={fr.profile.privacy} onPress={() => navigation.navigate('Privacy')} />
        </Card>
      </View>

      <View style={styles.group}>
        <Text variant="label" color="textMuted">
          {fr.profile.sectionAccount}
        </Text>
        <Card padded={false}>
          <ListRow
            title={fr.profile.notifications}
            onPress={() => navigation.navigate('Notifications')}
            right={<Badge count={unreadCount} />}
            testID="profile-notifications"
          />
          <Divider />
          <ListRow title={fr.profile.blocked} onPress={() => navigation.navigate('BlockedUsers')} />
          <Divider />
          <ListRow
            title={fr.profile.changePassword}
            onPress={() => navigation.navigate('ChangePassword')}
          />
          <Divider />
          <ListRow
            title={fr.profile.logout}
            hideChevron
            onPress={() => setConfirmLogout(true)}
            testID="profile-logout"
          />
          <Divider />
          <ListRow
            title={fr.profile.deleteAccount}
            danger
            onPress={() => navigation.navigate('DeleteAccount')}
          />
        </Card>
      </View>

      <ConfirmDialog
        visible={confirmLogout}
        title={fr.profile.logoutTitle}
        message={fr.profile.logoutMessage}
        confirmLabel={fr.profile.logout}
        loading={loggingOut}
        onConfirm={() => {
          setLoggingOut(true);
          logout().catch(() => {
            setLoggingOut(false);
            toast.error(fr.errors.unknown);
          });
        }}
        onCancel={() => setConfirmLogout(false)}
      />
    </Screen>
  );
}
