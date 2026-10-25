import { Animated, PanResponder, type PanResponderInstance } from "react-native";
import type { SwipeAction } from "../../api";

export type SwipeProps = {
  interactive: boolean;
  cardWidth: number;
  photoCount: number;
  onSwiped: (action: SwipeAction) => void;
  onOpenProfile: () => void;
  /** Tap on the left (-1) or right (+1) edge of a multi-photo card. */
  onPhotoStep: (delta: -1 | 1) => void;
};

export const SWIPE_DISTANCE_RATIO = 0.28;
export const SWIPE_VELOCITY = 0.8;
export const TAP_SLOP = 8;
export const EDGE_RATIO = 0.3;
const FLY_OUT_MS = 220;

export type Release = "like" | "pass" | "tap" | "cancel";

/** Pure decision taken when the finger lifts: fling/drag far enough, tap, or spring back. */
export function resolveRelease(gesture: { dx: number; dy: number; vx: number }, cardWidth: number): Release {
  if (Math.hypot(gesture.dx, gesture.dy) < TAP_SLOP) return "tap";
  if (gesture.dx > cardWidth * SWIPE_DISTANCE_RATIO || gesture.vx > SWIPE_VELOCITY) return "like";
  if (gesture.dx < -cardWidth * SWIPE_DISTANCE_RATIO || gesture.vx < -SWIPE_VELOCITY) return "pass";
  return "cancel";
}

/** What a tap at horizontal position `x` does: step through photos on the edges, open the profile elsewhere. */
export function resolveTap(x: number, cardWidth: number, photoCount: number): "previous" | "next" | "open" {
  if (photoCount > 1 && x < cardWidth * EDGE_RATIO) return "previous";
  if (photoCount > 1 && x > cardWidth * (1 - EDGE_RATIO)) return "next";
  return "open";
}

/** Owns the drag animation and gesture handling of one card; the React component only renders it. */
export class SwipeController {
  readonly position = new Animated.ValueXY();
  readonly responder: PanResponderInstance;
  private leaving = false;
  private props: SwipeProps;

  constructor(
    private readonly dragProgress: Animated.Value,
    initial: SwipeProps
  ) {
    this.props = initial;
    this.responder = PanResponder.create({
      onStartShouldSetPanResponder: () => this.props.interactive && !this.leaving,
      onMoveShouldSetPanResponder: (_event, gesture) =>
        this.props.interactive && !this.leaving && (Math.abs(gesture.dx) > TAP_SLOP || Math.abs(gesture.dy) > TAP_SLOP),
      onPanResponderMove: (_event, gesture) => {
        this.position.setValue({ x: gesture.dx, y: gesture.dy * 0.3 });
        this.dragProgress.setValue(Math.min(1, Math.abs(gesture.dx) / (this.props.cardWidth * 0.5)));
      },
      onPanResponderRelease: (event, gesture) => {
        const release = resolveRelease(gesture, this.props.cardWidth);
        if (release === "tap") {
          this.tap(event.nativeEvent.locationX);
        } else if (release === "cancel") {
          this.springBack();
        } else {
          this.swipe(release);
        }
      },
      onPanResponderTerminate: () => this.springBack()
    });
  }

  update(props: SwipeProps): void {
    this.props = props;
  }

  swipe(action: SwipeAction): void {
    if (this.leaving) {
      return;
    }
    this.leaving = true;
    const direction = action === "like" ? 1 : -1;
    Animated.timing(this.position, {
      toValue: { x: direction * this.props.cardWidth * 1.5, y: 40 },
      duration: FLY_OUT_MS,
      useNativeDriver: false
    }).start(() => this.props.onSwiped(action));
  }

  private tap(x: number): void {
    const target = resolveTap(x, this.props.cardWidth, this.props.photoCount);
    if (target === "open") {
      this.props.onOpenProfile();
    } else {
      this.props.onPhotoStep(target === "next" ? 1 : -1);
    }
  }

  private springBack(): void {
    this.dragProgress.setValue(0);
    Animated.spring(this.position, { toValue: { x: 0, y: 0 }, friction: 6, useNativeDriver: false }).start();
  }
}
