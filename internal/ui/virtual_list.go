package ui

import (
	"math"

	. "go.hasen.dev/shirei"
	. "go.hasen.dev/shirei/tw"
	. "go.hasen.dev/shirei/widgets"
)

func absF32(x float32) float32 {
	return float32(math.Abs(float64(x)))
}

func zeroIfNaN(a float32) float32 {
	if math.IsNaN(float64(a)) {
		return 0
	}
	return a
}

// VirtualListWithScroll is a tweaked VirtualListView that supports programmatic scrolling.
func VirtualListWithScroll(itemCount int, itemIdFn func(int) any, itemHeightFn ItemHeightFn, itemViewFn ItemViewFn, scrollToIndex int) {
	const N = 50

	type ItemOffset struct {
		Index  int
		Offset float32
	}

	type VirtualListState struct {
		Anchor            ItemOffset
		TotalHeight       float32
		ScrollOffset      float32
		Width             float32
		LastScrollToIndex int
	}

	computeAverageHeight := func(width float32) float32 {
		var topN int = min(N, itemCount)
		var seenHeight float32
		for i := 0; i < topN; i++ {
			seenHeight += max(1, itemHeightFn(i, width))
		}
		if topN == 0 {
			return 1 // fallback
		}
		return seenHeight / float32(topN)
	}

	itemOffsetFromAnchor := func(width float32, anchor ItemOffset, scrollOffset float32) ItemOffset {
		var result = anchor

		if scrollOffset < anchor.Offset {
			for result.Index > 0 {
				result.Index--
				result.Offset -= itemHeightFn(result.Index, width)
				if result.Offset <= scrollOffset {
					break
				}
			}
		} else {
			for result.Index < itemCount-1 {
				if result.Offset+itemHeightFn(result.Index, width) > scrollOffset {
					break
				}
				result.Offset += itemHeightFn(result.Index, width)
				result.Index++
			}
		}

		return result
	}

	anchorFromOffset := func(width float32, avgHeight float32, scrollOffset float32) ItemOffset {
		if itemCount <= N*2 {
			return itemOffsetFromAnchor(width, ItemOffset{}, scrollOffset)
		}

		var anchor ItemOffset
		anchor.Offset = float32(int(scrollOffset/avgHeight)) * avgHeight
		anchor.Index = int(zeroIfNaN(anchor.Offset / avgHeight))

		if anchor.Index <= N {
			return itemOffsetFromAnchor(width, ItemOffset{}, scrollOffset)
		} else if anchor.Index >= itemCount-N {
			var totalHeight = avgHeight * float32(itemCount)
			var offset = totalHeight
			for i := itemCount - 1; i >= anchor.Index; i-- {
				offset -= itemHeightFn(i, width)
			}
			anchor.Offset = offset
			return anchor
		} else {
			return anchor
		}
	}

	Layout(TW(Viewport, NoAnimate), func() {
		ScrollOnInput()
		ScrollBars()

		var widthChanged bool

		var state = Use[VirtualListState]("virtual-list-state")

		// First, check if we need to programmatically scroll to the selection
		size := GetResolvedSize()
		if scrollToIndex >= 0 && state.LastScrollToIndex != scrollToIndex && size[1] > 0 {
			state.LastScrollToIndex = scrollToIndex

			// Assume standard heights for simplicity or compute roughly where it is
			// Since our app uses fixed 60.0 height items, this works nicely, but let's be robust:
			targetY := float32(0)
			for i := 0; i < scrollToIndex; i++ {
				targetY += itemHeightFn(i, size[0])
			}
			itemH := itemHeightFn(scrollToIndex, size[0])

			viewportHeight := size[1]
			if targetY < state.ScrollOffset {
				SetScrollOffset(Vec2{0, targetY})
			} else if targetY+itemH > state.ScrollOffset+viewportHeight {
				SetScrollOffset(Vec2{0, targetY + itemH - viewportHeight})
			}
		}

		scroll := GetScrollOffset()

		width := max(0, size[0]-SCROLLBAR_WIDTH)
		if width <= 0 {
			RequestNextFrame()
			return
		}

		avgHeight := computeAverageHeight(width)

		var totalHeight0 = state.TotalHeight
		state.TotalHeight = avgHeight * float32(itemCount)

		var scrollOffset0 = state.ScrollOffset

		if width != state.Width {
			widthChanged = true
			state.Width = width
		}

		if scroll[1] != state.ScrollOffset {
			scrollAmount := absF32(state.ScrollOffset - scroll[1])
			state.ScrollOffset = scroll[1]

			var jumpThreshold = size[1] * 2

			if scrollAmount > jumpThreshold {
				state.Anchor = anchorFromOffset(width, avgHeight, state.ScrollOffset)
			}
		}

		if widthChanged && totalHeight0 > 0 {
			state.Anchor.Offset = zeroIfNaN(state.TotalHeight * state.Anchor.Offset / totalHeight0)
			state.ScrollOffset = zeroIfNaN(state.TotalHeight * scrollOffset0 / totalHeight0)
			SetScrollOffset(Vec2{0, state.ScrollOffset})
		}

		first := itemOffsetFromAnchor(width, state.Anchor, state.ScrollOffset)

		if first.Index == 0 {
			first.Offset = 0
		}
		if first.Offset < avgHeight && first.Index != 0 {
			first = itemOffsetFromAnchor(width, ItemOffset{}, state.ScrollOffset)
		}

		state.Anchor = first

		spaceBefore := first.Offset

		Element(TW(FixHeight(spaceBefore)))

		var renderedHeight = -(state.ScrollOffset - spaceBefore)

		var startIndex int = first.Index
		var endIndex int = itemCount

		for idx := startIndex; idx < itemCount; idx++ {
			endIndex = idx + 1
			height := itemHeightFn(idx, width)
			renderedHeight += height

			id := itemIdFn(idx)
			LayoutId(id, TW(FixSize(width, height)), func() {
				itemViewFn(idx, width)
			})

			if renderedHeight > size[1] {
				break
			}
		}

		spaceAfter := max(0, state.TotalHeight-(spaceBefore+renderedHeight))

		if endIndex == itemCount {
			spaceAfter = 0
		}
		if endIndex != itemCount && spaceAfter < avgHeight {
			remainingCount := itemCount - endIndex
			spaceAfter = float32(remainingCount) * avgHeight
		}

		Element(TW(FixHeight(spaceAfter)))
	})
}
