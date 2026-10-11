import { useEffect, useRef, useState } from 'react';
import type { ReactNode } from 'react';
import { Animated, ScrollView, View } from 'react-native';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { errorMessage, isNotFound, profileApi, queryKeys } from '../../api';
import type { OwnProfile } from '../../api';
import { fr } from '../../i18n';
import { useErrorToast } from '../../lib/useErrorToast';
import { parseBirthDate } from '../../lib/validation';
import { useStyles } from '../../theme';
import { Button, ErrorState, Loading, Screen, Text, useToast } from '../../ui';
import { useAuth } from '../auth/AuthContext';
import {
  BioInput,
  BirthDateInput,
  FirstNameInput,
  GenderPicker,
} from '../profile/components/ProfileFields';
import { InterestPicker } from '../profile/components/InterestPicker';
import { LocationField } from '../profile/components/LocationField';
import { PhotoManager } from '../profile/components/PhotoManager';
import { PreferencesFields } from '../profile/components/PreferencesFields';
import { useOwnProfile } from '../profile/hooks';
import {
  ONBOARDING_STEPS,
  deriveInitialStep,
  emptyForm,
  formFromProfile,
  toProfileInput,
  validateStep,
} from './logic';
import type { FieldErrors, OnboardingForm, OnboardingStep } from './logic';

/** Loads the current profile (404 = nothing saved yet) and derives where to resume. */
export function OnboardingScreen() {
  const profile = useOwnProfile();
  if (profile.isPending) return <Loading />;
  if (profile.isError && !isNotFound(profile.error)) {
    return (
      <Screen edges={['top', 'bottom']}>
        <ErrorState message={errorMessage(profile.error)} onRetry={() => void profile.refetch()} />
      </Screen>
    );
  }
  return <Wizard existing={profile.data ?? null} />;
}

function Wizard({ existing }: { existing: OwnProfile | null }) {
  const qc = useQueryClient();
  const toast = useToast();
  const onError = useErrorToast();
  const { logout, refreshUser } = useAuth();
  const [step, setStep] = useState<OnboardingStep>(() => deriveInitialStep(existing));
  const [form, setForm] = useState<OnboardingForm>(() =>
    existing ? formFromProfile(existing) : emptyForm(),
  );
  const [errors, setErrors] = useState<FieldErrors>({});
  const [leaving, setLeaving] = useState(false);
  const progress = useRef(new Animated.Value(0)).current;
  const photosProfile = useOwnProfile();

  const index = ONBOARDING_STEPS.indexOf(step);
  const styles = useStyles((t) => ({
    top: {
      flexDirection: 'row',
      alignItems: 'center',
      justifyContent: 'space-between',
      paddingTop: t.spacing.sm,
    },
    track: {
      height: 6,
      borderRadius: 3,
      backgroundColor: t.colors.surfaceAlt,
      overflow: 'hidden',
      marginTop: t.spacing.sm,
    },
    fill: { height: 6, borderRadius: 3, backgroundColor: t.colors.primary },
    body: { paddingVertical: t.spacing.xl, gap: t.spacing.xl },
    footer: { paddingVertical: t.spacing.lg, gap: t.spacing.sm },
    head: { gap: t.spacing.sm },
  }));

  useEffect(() => {
    Animated.timing(progress, {
      toValue: (index + 1) / ONBOARDING_STEPS.length,
      duration: 260,
      useNativeDriver: false,
    }).start();
  }, [index, progress]);

  const patch = (change: Partial<OnboardingForm>) => {
    setForm((f) => ({ ...f, ...change }));
    setErrors({});
  };

  const save = useMutation({
    mutationFn: async () => {
      const input = toProfileInput(form);
      if (!input) throw new Error('invalid_form');
      await profileApi.save(input);
      await profileApi.savePreferences(form.preferences);
      if (form.coordinates) {
        try {
          await profileApi.setLocation(form.coordinates);
        } catch {
          toast.error(fr.location.saveFailed);
        }
      }
    },
    onSuccess: async () => {
      await qc.invalidateQueries({ queryKey: queryKeys.ownProfile });
      setStep('photos');
    },
    onError,
  });

  const finish = useMutation({
    mutationFn: async () => {
      const me = await refreshUser();
      if (!me || !me.profileComplete) throw new Error('profile_incomplete');
    },
    onError: () => toast.error(fr.onboarding.finishFailed),
  });

  const next = () => {
    const found = validateStep(step, form);
    setErrors(found);
    if (Object.keys(found).length > 0) return;
    if (step === 'interests') {
      save.mutate();
      return;
    }
    setStep(ONBOARDING_STEPS[index + 1]);
  };
  const back = () => {
    setErrors({});
    if (index > 0) setStep(ONBOARDING_STEPS[index - 1]);
  };

  const birth = parseBirthDate(form.birth.day, form.birth.month, form.birth.year);
  const photoCount = photosProfile.data?.photos.length ?? existing?.photos.length ?? 0;
  const lastStep = step === 'photos';

  let content: ReactNode;
  switch (step) {
    case 'identity':
      content = (
        <>
          <FirstNameInput
            value={form.firstName}
            onChange={(firstName) => patch({ firstName })}
            error={errors.firstName}
          />
          <BirthDateInput
            value={form.birth}
            onChange={(value) => patch({ birth: value })}
            error={errors.birth}
            disabled={existing !== null}
          />
          {birth.ok ? (
            <Text color="success" weight="600">
              {fr.onboarding.ageOk(birth.age)}
            </Text>
          ) : null}
          <GenderPicker
            value={form.gender}
            onChange={(gender) => patch({ gender })}
            error={errors.gender}
          />
        </>
      );
      break;
    case 'preferences':
      content = (
        <PreferencesFields
          value={form.preferences}
          onChange={(preferences) => patch({ preferences })}
          interestedInError={errors.interestedIn}
        />
      );
      break;
    case 'location':
      content = (
        <LocationField
          city={form.city}
          onCityChange={(city) => patch({ city })}
          cityError={errors.city}
          hasLocation={form.hasLocation}
          coordinates={form.coordinates}
          mode="deferred"
          onCoordinates={(coordinates) => patch({ coordinates })}
        />
      );
      break;
    case 'bio':
      content = <BioInput value={form.bio} onChange={(bio) => patch({ bio })} />;
      break;
    case 'interests':
      content = (
        <InterestPicker
          selectedIds={form.interestIds}
          onChange={(interestIds) => patch({ interestIds })}
        />
      );
      break;
    case 'photos':
      content = photosProfile.isPending ? (
        <Loading />
      ) : photosProfile.isError ? (
        <ErrorState
          message={errorMessage(photosProfile.error)}
          onRetry={() => void photosProfile.refetch()}
        />
      ) : (
        <PhotoManager photos={photosProfile.data.photos} />
      );
      break;
  }

  return (
    <Screen edges={['top', 'bottom']}>
      <View>
        <View style={styles.top}>
          {index > 0 ? (
            <Button
              label={fr.common.back}
              variant="ghost"
              size="sm"
              fullWidth={false}
              onPress={back}
            />
          ) : (
            <View />
          )}
          <Button
            label={fr.onboarding.logout}
            variant="ghost"
            size="sm"
            fullWidth={false}
            loading={leaving}
            onPress={() => {
              setLeaving(true);
              void logout();
            }}
          />
        </View>
        <View
          style={styles.track}
          accessible
          accessibilityRole="progressbar"
          accessibilityLabel={fr.onboarding.progress(index + 1, ONBOARDING_STEPS.length)}
        >
          <Animated.View
            style={[
              styles.fill,
              { width: progress.interpolate({ inputRange: [0, 1], outputRange: ['0%', '100%'] }) },
            ]}
          />
        </View>
      </View>

      <ScrollView
        keyboardShouldPersistTaps="handled"
        contentContainerStyle={styles.body}
        style={{ flex: 1 }}
      >
        <View style={styles.head}>
          <Text variant="caption" color="primary" weight="700">
            {fr.onboarding.progress(index + 1, ONBOARDING_STEPS.length)}
          </Text>
          <Text variant="title">{fr.onboarding.steps[step].title}</Text>
          <Text color="textMuted">{fr.onboarding.steps[step].subtitle}</Text>
        </View>
        {content}
      </ScrollView>

      <View style={styles.footer}>
        {lastStep ? (
          <>
            {photoCount < 1 ? (
              <Text variant="small" color="textMuted" align="center">
                {fr.onboarding.photoRequired}
              </Text>
            ) : null}
            <Button
              label={fr.onboarding.finish}
              onPress={() => finish.mutate()}
              disabled={photoCount < 1}
              loading={finish.isPending}
              testID="onboarding-finish"
            />
          </>
        ) : (
          <Button
            label={step === 'interests' ? fr.onboarding.saveAndContinue : fr.common.continue}
            onPress={next}
            loading={save.isPending}
            testID="onboarding-next"
          />
        )}
      </View>
    </Screen>
  );
}
