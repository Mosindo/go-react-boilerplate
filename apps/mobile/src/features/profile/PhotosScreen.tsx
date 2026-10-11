import { errorMessage } from '../../api';
import { fr } from '../../i18n';
import { ErrorState, Loading, Screen, Text } from '../../ui';
import { PhotoManager } from './components/PhotoManager';
import { useOwnProfile } from './hooks';

export function PhotosScreen() {
  const profile = useOwnProfile();
  if (profile.isPending) return <Loading />;
  if (profile.isError) {
    return (
      <Screen edges={['bottom']}>
        <ErrorState message={errorMessage(profile.error)} onRetry={() => void profile.refetch()} />
      </Screen>
    );
  }
  return (
    <Screen scroll edges={['bottom']}>
      <Text color="textMuted">{fr.photos.intro}</Text>
      <PhotoManager photos={profile.data.photos} />
    </Screen>
  );
}
