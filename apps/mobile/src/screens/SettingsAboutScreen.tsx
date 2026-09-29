import React from "react";
import { View } from "react-native";
import appConfig from "../../app.json";
import { strings } from "../lib/strings";
import { Section } from "../shared/layout/Section";
import { Card } from "../shared/ui/Card";
import { Notice } from "../shared/ui/Notice";
import { Text } from "../shared/ui/Text";
import { type Theme } from "../shared/ui/theme";
import { useThemedStyles } from "../shared/ui/useThemedStyles";
import { ProfileSubScreen } from "./ProfileSubScreen";

export const APP_VERSION: string = appConfig.expo.version;

const makeStyles = (t: Theme) => ({ tips: { gap: t.spacing.sm } });

export function SettingsAboutScreen({ onBack }: { onBack: () => void }) {
  const styles = useThemedStyles(makeStyles);
  return (
    <ProfileSubScreen onBack={onBack} testID="settings-about-screen" title="About Alba">
      <Notice title={strings.freePromise} tone="success" />
      <Section subtitle="Small habits that keep you safe." title="Safety tips">
        <Card>
          <View style={styles.tips}>
            {strings.profile.safetyTips.map((tip) => (
              <Text key={tip}>{`•  ${tip}`}</Text>
            ))}
          </View>
        </Card>
      </Section>
      <Section title="Report and block">
        <Text tone="muted">
          You can block or report anyone from their profile or from a chat. Blocked people can no longer see you or
          message you. Reports are reviewed by a person.
        </Text>
      </Section>
      <Text tone="muted" variant="caption">
        Alba version {APP_VERSION}
      </Text>
    </ProfileSubScreen>
  );
}
