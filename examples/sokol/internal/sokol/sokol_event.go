package sokol

const (
	SappMaxTouchpoints = 8
)

type SappEventType int32

const (
	SappEventTypeInvalid SappEventType = iota
	SappEventTypeKeyDown
	SappEventTypeKeyUp
	SappEventTypeChar
	SappEventTypeMouseDown
	SappEventTypeMouseUp
	SappEventTypeMouseScroll
	SappEventTypeMouseMove
	SappEventTypeMouseEnter
	SappEventTypeMouseLeave
	SappEventTypeTouchesBegan
	SappEventTypeTouchesMoved
	SappEventTypeTouchesEnded
	SappEventTypeTouchesCancelled
	SappEventTypeResized
	SappEventTypeIconified
	SappEventTypeRestored
	SappEventTypeFocused
	SappEventTypeUnfocused
	SappEventTypeSuspended
	SappEventTypeResumed
	SappEventTypeQuitRequested
	SappEventTypeClipboardPasted
	SappEventTypeFilesDropped
	SappEventTypeNum
	SappEventTypeForceU32 SappEventType = 0x7FFFFFFF
)

type SappKeycode int32

const (
	SappKeycodeInvalid      SappKeycode = iota
	SappKeycodeSpace        SappKeycode = 32
	SappKeycodeApostrophe   SappKeycode = 39
	SappKeycodeComma        SappKeycode = 44
	SappKeycodeMinus        SappKeycode = 45
	SappKeycodePeriod       SappKeycode = 46
	SappKeycodeSlash        SappKeycode = 47
	SappKeycode0            SappKeycode = 48
	SappKeycode1            SappKeycode = 49
	SappKeycode2            SappKeycode = 50
	SappKeycode3            SappKeycode = 51
	SappKeycode4            SappKeycode = 52
	SappKeycode5            SappKeycode = 53
	SappKeycode6            SappKeycode = 54
	SappKeycode7            SappKeycode = 55
	SappKeycode8            SappKeycode = 56
	SappKeycode9            SappKeycode = 57
	SappKeycodeSemicolon    SappKeycode = 59
	SappKeycodeEqual        SappKeycode = 61
	SappKeycodeA            SappKeycode = 65
	SappKeycodeB            SappKeycode = 66
	SappKeycodeC            SappKeycode = 67
	SappKeycodeD            SappKeycode = 68
	SappKeycodeE            SappKeycode = 69
	SappKeycodeF            SappKeycode = 70
	SappKeycodeG            SappKeycode = 71
	SappKeycodeH            SappKeycode = 72
	SappKeycodeI            SappKeycode = 73
	SappKeycodeJ            SappKeycode = 74
	SappKeycodeK            SappKeycode = 75
	SappKeycodeL            SappKeycode = 76
	SappKeycodeM            SappKeycode = 77
	SappKeycodeN            SappKeycode = 78
	SappKeycodeO            SappKeycode = 79
	SappKeycodeP            SappKeycode = 80
	SappKeycodeQ            SappKeycode = 81
	SappKeycodeR            SappKeycode = 82
	SappKeycodeS            SappKeycode = 83
	SappKeycodeT            SappKeycode = 84
	SappKeycodeU            SappKeycode = 85
	SappKeycodeV            SappKeycode = 86
	SappKeycodeW            SappKeycode = 87
	SappKeycodeX            SappKeycode = 88
	SappKeycodeY            SappKeycode = 89
	SappKeycodeZ            SappKeycode = 90
	SappKeycodeLeftBracket  SappKeycode = 91
	SappKeycodeBackslash    SappKeycode = 92
	SappKeycodeRightBracket SappKeycode = 93
	SappKeycodeGraveAccent  SappKeycode = 96
	SappKeycodeWorld1       SappKeycode = 161
	SappKeycodeWorld2       SappKeycode = 162
	SappKeycodeEscape       SappKeycode = 256
	SappKeycodeEnter        SappKeycode = 257
	SappKeycodeTab          SappKeycode = 258
	SappKeycodeBackspace    SappKeycode = 259
	SappKeycodeInsert       SappKeycode = 260
	SappKeycodeDelete       SappKeycode = 261
	SappKeycodeRight        SappKeycode = 262
	SappKeycodeLeft         SappKeycode = 263
	SappKeycodeDown         SappKeycode = 264
	SappKeycodeUp           SappKeycode = 265
	SappKeycodePageUp       SappKeycode = 266
	SappKeycodePageDown     SappKeycode = 267
	SappKeycodeHome         SappKeycode = 268
	SappKeycodeEnd          SappKeycode = 269
	SappKeycodeCapsLock     SappKeycode = 280
	SappKeycodeScrollLock   SappKeycode = 281
	SappKeycodeNumLock      SappKeycode = 282
	SappKeycodePrintScreen  SappKeycode = 283
	SappKeycodePause        SappKeycode = 284
	SappKeycodeF1           SappKeycode = 290
	SappKeycodeF2           SappKeycode = 291
	SappKeycodeF3           SappKeycode = 292
	SappKeycodeF4           SappKeycode = 293
	SappKeycodeF5           SappKeycode = 294
	SappKeycodeF6           SappKeycode = 295
	SappKeycodeF7           SappKeycode = 296
	SappKeycodeF8           SappKeycode = 297
	SappKeycodeF9           SappKeycode = 298
	SappKeycodeF10          SappKeycode = 299
	SappKeycodeF11          SappKeycode = 300
	SappKeycodeF12          SappKeycode = 301
	SappKeycodeF13          SappKeycode = 302
	SappKeycodeF14          SappKeycode = 303
	SappKeycodeF15          SappKeycode = 304
	SappKeycodeF16          SappKeycode = 305
	SappKeycodeF17          SappKeycode = 306
	SappKeycodeF18          SappKeycode = 307
	SappKeycodeF19          SappKeycode = 308
	SappKeycodeF20          SappKeycode = 309
	SappKeycodeF21          SappKeycode = 310
	SappKeycodeF22          SappKeycode = 311
	SappKeycodeF23          SappKeycode = 312
	SappKeycodeF24          SappKeycode = 313
	SappKeycodeF25          SappKeycode = 314
	SappKeycodeKP0          SappKeycode = 320
	SappKeycodeKP1          SappKeycode = 321
	SappKeycodeKP2          SappKeycode = 322
	SappKeycodeKP3          SappKeycode = 323
	SappKeycodeKP4          SappKeycode = 324
	SappKeycodeKP5          SappKeycode = 325
	SappKeycodeKP6          SappKeycode = 326
	SappKeycodeKP7          SappKeycode = 327
	SappKeycodeKP8          SappKeycode = 328
	SappKeycodeKP9          SappKeycode = 329
	SappKeycodeKPDecimal    SappKeycode = 330
	SappKeycodeKPDivide     SappKeycode = 331
	SappKeycodeKPMultiply   SappKeycode = 332
	SappKeycodeKPSubtract   SappKeycode = 333
	SappKeycodeKPAdd        SappKeycode = 334
	SappKeycodeKPEnter      SappKeycode = 335
	SappKeycodeKPEqual      SappKeycode = 336
	SappKeycodeLeftShift    SappKeycode = 340
	SappKeycodeLeftControl  SappKeycode = 341
	SappKeycodeLeftAlt      SappKeycode = 342
	SappKeycodeLeftSuper    SappKeycode = 343
	SappKeycodeRightShift   SappKeycode = 344
	SappKeycodeRightControl SappKeycode = 345
	SappKeycodeRightAlt     SappKeycode = 346
	SappKeycodeRightSuper   SappKeycode = 347
	SappKeycodeMenu         SappKeycode = 348
)

type SappMouseButton int32

const (
	SappMouseButtonLeft   SappMouseButton = 0x0
	SappMouseButtonRight  SappMouseButton = 0x1
	SappMouseButtonMiddle SappMouseButton = 0x2
	SappMouseButtonNum    SappMouseButton = 0x100
)

type SappAndroidToolType int32

const (
	SappAndroidTollTypeUnknown SappAndroidToolType = iota
	SappAndroidToolTypeFinger
	SappAndroidToolTypeStylus
	SappAndroidToolTypeMouse
)

type SappTouchPoint struct {
	Identifier      uint64
	PosX            float32
	PosY            float32
	AndroidToolType SappAndroidToolType
	Changed         bool
}

type SappEvent struct {
	FrameCount        uint64
	Type              SappEventType
	KeyCode           SappKeycode
	CharCode          uint32
	KeyRepeat         bool
	Mofifiers         uint32
	MouseButton       SappMouseButton
	MouseX            float32
	MouseY            float32
	MouseDx           float32
	MouseDy           float32
	ScrollX           float32
	ScrollY           float32
	NumTouches        int32
	Touches           [SappMaxTouchpoints]SappTouchPoint
	WindowWidth       int32
	WindowHeight      int32
	FramebufferWidth  int32
	FramebufferHeight int32
}
