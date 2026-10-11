import { Text as RNText } from 'react-native';
import type { TextProps as RNTextProps, TextStyle } from 'react-native';
import { useStyles, useTheme } from '../theme';
import type { ColorTokens } from '../theme';

export type TextVariant =
  'display' | 'title' | 'heading' | 'lead' | 'body' | 'small' | 'caption' | 'label';

export type TextColor = keyof Pick<
  ColorTokens,
  | 'text'
  | 'textMuted'
  | 'primary'
  | 'secondary'
  | 'danger'
  | 'success'
  | 'textOnPrimary'
  | 'textOnSecondary'
  | 'onPhoto'
>;

export interface TextProps extends RNTextProps {
  variant?: TextVariant;
  color?: TextColor;
  align?: TextStyle['textAlign'];
  weight?: TextStyle['fontWeight'];
}

export function Text({
  variant = 'body',
  color = 'text',
  align,
  weight,
  style,
  accessibilityRole,
  ...rest
}: TextProps) {
  const { colors } = useTheme();
  const styles = useStyles((t) => ({
    display: {
      fontSize: t.fontSizes.display,
      lineHeight: 42,
      fontWeight: '800',
      letterSpacing: -0.5,
    },
    title: { fontSize: t.fontSizes.title, lineHeight: 34, fontWeight: '800', letterSpacing: -0.3 },
    heading: { fontSize: t.fontSizes.heading, lineHeight: 28, fontWeight: '700' },
    lead: { fontSize: t.fontSizes.lead, lineHeight: 26, fontWeight: '400' },
    body: { fontSize: t.fontSizes.body, lineHeight: 23, fontWeight: '400' },
    small: { fontSize: t.fontSizes.small, lineHeight: 20, fontWeight: '400' },
    caption: { fontSize: t.fontSizes.caption, lineHeight: 18, fontWeight: '400' },
    label: { fontSize: t.fontSizes.small, lineHeight: 20, fontWeight: '600' },
  }));
  const isHeading = variant === 'display' || variant === 'title' || variant === 'heading';
  return (
    <RNText
      accessibilityRole={accessibilityRole ?? (isHeading ? 'header' : undefined)}
      style={[
        styles[variant],
        { color: colors[color] },
        align && { textAlign: align },
        weight && { fontWeight: weight },
        style,
      ]}
      {...rest}
    />
  );
}
